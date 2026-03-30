package processor

import (
	"context"
	"log/slog"
	"sync"
)

// ConcurrentProcessor processes batches concurrently
type ConcurrentProcessor struct {
	batchProcessor *BatchProcessor
	concurrency    int
}

// NewConcurrentProcessor creates a new concurrent processor
func NewConcurrentProcessor(bp *BatchProcessor, concurrency int) *ConcurrentProcessor {
	return &ConcurrentProcessor{
		batchProcessor: bp,
		concurrency:    concurrency,
	}
}

// ProcessAll processes all batches with controlled concurrency.
// When streaming is enabled, batches are processed sequentially to avoid
// interleaved output. Otherwise, batches are processed concurrently.
func (p *ConcurrentProcessor) ProcessAll(ctx context.Context, batches []*Batch) error {
	// Sequential processing when streaming to avoid interleaved output
	if p.batchProcessor.HasStreamWriter() {
		return p.processSequential(ctx, batches)
	}
	return p.processConcurrent(ctx, batches)
}

// processSequential processes batches one at a time
func (p *ConcurrentProcessor) processSequential(ctx context.Context, batches []*Batch) error {
	var failedCount int
	var failedErrors []error

	for i, batch := range batches {
		if err := p.batchProcessor.ProcessBatch(ctx, batch); err != nil {
			failedCount++
			failedErrors = append(failedErrors, err)
			slog.Warn("batch processing failed",
				"batch_index", i,
				"batch_type", batch.Type,
				"error", err)
		}
	}

	if failedCount > 0 {
		return &PartialEnrichmentError{
			TotalBatches:  len(batches),
			FailedBatches: failedCount,
			Errors:        failedErrors,
		}
	}
	return nil
}

// processConcurrent processes batches with controlled concurrency
func (p *ConcurrentProcessor) processConcurrent(ctx context.Context, batches []*Batch) error {
	var (
		wg           sync.WaitGroup
		mu           sync.Mutex
		failedCount  int
		failedErrors []error
	)

	semaphore := make(chan struct{}, p.concurrency)

	for i, batch := range batches {
		wg.Add(1)

		go func(idx int, b *Batch) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := p.batchProcessor.ProcessBatch(ctx, b); err != nil {
				mu.Lock()
				failedCount++
				failedErrors = append(failedErrors, err)
				mu.Unlock()

				slog.Warn("batch processing failed",
					"batch_index", idx,
					"batch_type", b.Type,
					"error", err)
			}
		}(i, batch)
	}

	wg.Wait()

	// Return partial error if some batches failed
	if failedCount > 0 {
		return &PartialEnrichmentError{
			TotalBatches:  len(batches),
			FailedBatches: failedCount,
			Errors:        failedErrors,
		}
	}

	return nil
}
