package processor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/schollz/progressbar/v3"

	"github.com/spencercjh/spec-forge/internal/enricher/provider"
)

// ConcurrentProcessor processes batches with configurable concurrency.
// Note: When streaming is enabled (StreamWriter configured), processing is
// sequential to ensure readable interleaved output. The concurrency setting
// only applies when streaming is disabled via --no-stream.
type ConcurrentProcessor struct {
	batchProcessor *BatchProcessor
	concurrency    int
}

// NewConcurrentProcessor creates a new concurrent processor.
// The concurrency parameter is only used when streaming is disabled.
func NewConcurrentProcessor(bp *BatchProcessor, concurrency int) *ConcurrentProcessor {
	return &ConcurrentProcessor{
		batchProcessor: bp,
		concurrency:    concurrency,
	}
}

// ProcessAll processes all batches.
// When streaming is enabled, batches are processed sequentially to prevent
// interleaved LLM output chunks from mixing together (which would be unreadable).
// When streaming is disabled (--no-stream), batches are processed concurrently
// up to the configured concurrency limit for faster processing.
// Returns the accumulated token usage across all batches.
func (p *ConcurrentProcessor) ProcessAll(ctx context.Context, batches []*Batch) (*provider.TokenUsage, error) {
	// Sequential processing when streaming to avoid interleaved output
	if p.batchProcessor.HasStreamWriter() {
		return p.processSequential(ctx, batches)
	}
	return p.processConcurrent(ctx, batches)
}

// processSequential processes batches one at a time (streaming mode).
// This ensures LLM output chunks are grouped by batch type for readability.
func (p *ConcurrentProcessor) processSequential(ctx context.Context, batches []*Batch) (*provider.TokenUsage, error) {
	var (
		totalUsage   provider.TokenUsage
		failedCount  int
		failedErrors []error
	)

	bar := progressbar.NewOptions(len(batches),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("Enriching OpenAPI spec..."),
		progressbar.OptionOnCompletion(func() { fmt.Fprint(os.Stderr, "\n") }),
	)

	for i, batch := range batches {
		usage, err := p.batchProcessor.ProcessBatch(ctx, batch)
		if usage != nil {
			totalUsage.Add(usage)
		}
		if err != nil {
			failedCount++
			failedErrors = append(failedErrors, err)
			bar.Describe(fmt.Sprintf("Enriching... | %d failed", failedCount))
			slog.Warn("batch processing failed",
				"batch_index", i,
				"batch_type", batch.Type,
				"error", err)
		}
		if err := bar.Add(1); err != nil {
			slog.Debug("progress bar add failed", "error", err)
		}
	}

	if err := bar.Finish(); err != nil {
		slog.Debug("progress bar finish failed", "error", err)
	}

	if failedCount > 0 {
		return &totalUsage, &PartialEnrichmentError{
			TotalBatches:  len(batches),
			FailedBatches: failedCount,
			Errors:        failedErrors,
		}
	}
	return &totalUsage, nil
}

// processConcurrent processes batches with controlled concurrency.
func (p *ConcurrentProcessor) processConcurrent(ctx context.Context, batches []*Batch) (*provider.TokenUsage, error) {
	var (
		wg           sync.WaitGroup
		mu           sync.Mutex
		totalUsage   provider.TokenUsage
		failedCount  int
		failedErrors []error
	)

	bar := progressbar.NewOptions(len(batches),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("Enriching OpenAPI spec..."),
		progressbar.OptionOnCompletion(func() { fmt.Fprint(os.Stderr, "\n") }),
	)

	semaphore := make(chan struct{}, p.concurrency)

	for i, batch := range batches {
		wg.Add(1)

		go func(idx int, b *Batch) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			usage, err := p.batchProcessor.ProcessBatch(ctx, b)
			mu.Lock()
			if usage != nil {
				totalUsage.Add(usage)
			}
			if err != nil {
				failedCount++
				failedErrors = append(failedErrors, err)
				bar.Describe(fmt.Sprintf("Enriching... | %d failed", failedCount))
				slog.Warn("batch processing failed",
					"batch_index", idx,
					"batch_type", b.Type,
					"error", err)
			}
			mu.Unlock()
			if err := bar.Add(1); err != nil {
				slog.Debug("progress bar add failed", "error", err)
			}
		}(i, batch)
	}

	wg.Wait()
	if err := bar.Finish(); err != nil {
		slog.Debug("progress bar finish failed", "error", err)
	}

	if failedCount > 0 {
		return &totalUsage, &PartialEnrichmentError{
			TotalBatches:  len(batches),
			FailedBatches: failedCount,
			Errors:        failedErrors,
		}
	}

	return &totalUsage, nil
}
