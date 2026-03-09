// Package spring tests the Spring extractor implementation.
package spring

import (
	"os"
	"testing"

	"github.com/vifraa/gopom"
)

const (
	mavenTestPath = "../../../integration-tests/maven-springboot-openapi-demo/pom.xml"
)

func TestMavenParser_Parse(t *testing.T) {
	pomPath := mavenTestPath

	// Skip if project doesn't exist
	if _, err := os.Stat(pomPath); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	parser := newMavenParser()
	pom, err := parser.Parse(pomPath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if pom == nil {
		t.Fatal("Parsed pom should not be nil")
	}
}

func TestMavenParser_GetSpringBootVersion(t *testing.T) {
	pomPath := mavenTestPath

	if _, err := os.Stat(pomPath); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	parser := newMavenParser()
	pom, err := parser.Parse(pomPath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	version := parser.GetSpringBootVersion(pom)
	if version == "" {
		t.Error("Spring Boot version should not be empty")
	}
	// The demo project uses Spring Boot 4.0.3
	if version != testSpringBootVersion {
		t.Logf("Warning: Expected Spring Boot %s, got %s", testSpringBootVersion, version)
	}
}

func TestMavenParser_HasSpringdocDependency(t *testing.T) {
	pomPath := mavenTestPath

	if _, err := os.Stat(pomPath); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	parser := newMavenParser()
	pom, err := parser.Parse(pomPath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !parser.HasSpringdocDependency(pom) {
		t.Error("HasSpringdocDependency should be true for demo project")
	}
}

func TestMavenParser_GetSpringdocVersion(t *testing.T) {
	pomPath := mavenTestPath

	if _, err := os.Stat(pomPath); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	parser := newMavenParser()
	pom, err := parser.Parse(pomPath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	version := parser.GetSpringdocVersion(pom)
	if version == "" {
		t.Error("Springdoc version should not be empty for demo project")
	}
}

func TestMavenParser_HasSpringdocPlugin(t *testing.T) {
	pomPath := mavenTestPath

	if _, err := os.Stat(pomPath); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	parser := newMavenParser()
	pom, err := parser.Parse(pomPath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !parser.HasSpringdocPlugin(pom) {
		t.Error("HasSpringdocPlugin should be true for demo project")
	}
}

func TestMavenParser_FindDependency(t *testing.T) {
	parser := newMavenParser()

	// Create a minimal pom for testing
	webVersion := "4.0.3"
	deps := []gopom.Dependency{
		{
			GroupID:    new("org.springframework.boot"),
			ArtifactID: new("spring-boot-starter-web"),
			Version:    &webVersion,
		},
	}
	pom := &gopom.Project{
		Dependencies: &deps,
	}

	dep := parser.FindDependency(pom, "org.springframework.boot", "spring-boot-starter-web")
	if dep == nil {
		t.Fatal("Should find spring-boot-starter-web dependency")
	}
	if *dep.Version != "4.0.3" {
		t.Errorf("Version = %s, want 4.0.3", *dep.Version)
	}
}

func TestMavenParser_AddPlugin(t *testing.T) {
	parser := newMavenParser()

	// Create a minimal pom without plugins
	pom := &gopom.Project{}

	parser.AddPlugin(pom, "org.springdoc", "springdoc-openapi-maven-plugin", "1.5")

	if pom.Build == nil || pom.Build.Plugins == nil {
		t.Fatal("Build or Plugins should be set")
	}

	plugins := *pom.Build.Plugins
	if len(plugins) != 1 {
		t.Fatalf("Expected 1 plugin, got %d", len(plugins))
	}

	plugin := plugins[0]
	if *plugin.GroupID != "org.springdoc" {
		t.Errorf("Expected GroupID org.springdoc, got %s", *plugin.GroupID)
	}
	if *plugin.ArtifactID != "springdoc-openapi-maven-plugin" {
		t.Errorf("Expected ArtifactID springdoc-openapi-maven-plugin, got %s", *plugin.ArtifactID)
	}
	if *plugin.Version != "1.5" {
		t.Errorf("Expected Version 1.5, got %s", *plugin.Version)
	}

	// Check execution configuration
	if plugin.Executions == nil || len(*plugin.Executions) != 1 {
		t.Fatalf("Expected 1 execution, got %d", len(*plugin.Executions))
	}

	exec := (*plugin.Executions)[0]
	if exec.ID == nil || *exec.ID != "integration-test" {
		t.Errorf("Expected execution ID 'integration-test', got %v", exec.ID)
	}
	if exec.Phase == nil || *exec.Phase != "integration-test" {
		t.Errorf("Expected execution phase 'integration-test', got %v", exec.Phase)
	}
	if exec.Goals == nil || len(*exec.Goals) != 1 || (*exec.Goals)[0] != "generate" {
		t.Errorf("Expected execution goals ['generate'], got %v", exec.Goals)
	}
}

func TestMavenParser_HasSpringBootStartStopGoals(t *testing.T) {
	parser := newMavenParser()

	t.Run("has start/stop goals", func(t *testing.T) {
		pom, err := parser.Parse(mavenTestPath)
		if err != nil {
			t.Fatalf("Failed to parse pom.xml: %v", err)
		}

		// The test pom.xml has spring-boot-maven-plugin with start/stop goals
		if !parser.HasSpringBootStartStopGoals(pom) {
			t.Error("Expected HasSpringBootStartStopGoals to return true")
		}
	})

	t.Run("no spring-boot plugin", func(t *testing.T) {
		pom := &gopom.Project{}
		if parser.HasSpringBootStartStopGoals(pom) {
			t.Error("Expected HasSpringBootStartStopGoals to return false for empty project")
		}
	})
}

func TestMavenParser_ConfigureSpringBootPlugin(t *testing.T) {
	parser := newMavenParser()

	t.Run("already configured", func(t *testing.T) {
		pom, err := parser.Parse(mavenTestPath)
		if err != nil {
			t.Fatalf("Failed to parse pom.xml: %v", err)
		}

		// Should not modify since it already has start/stop
		modified := parser.ConfigureSpringBootPlugin(pom)
		if modified {
			t.Error("Expected ConfigureSpringBootPlugin to return false for already configured plugin")
		}
	})

	t.Run("add start/stop goals", func(t *testing.T) {
		// Create a pom with spring-boot plugin but no start/stop goals
		groupID := "org.springframework.boot"
		artifactID := "spring-boot-maven-plugin"
		plugins := []gopom.Plugin{
			{
				GroupID:    &groupID,
				ArtifactID: &artifactID,
			},
		}
		pom := &gopom.Project{
			Build: &gopom.Build{},
		}
		pom.Build.Plugins = &plugins

		modified := parser.ConfigureSpringBootPlugin(pom)
		if !modified {
			t.Error("Expected ConfigureSpringBootPlugin to return true")
		}

		// Verify start/stop goals were added
		if !parser.HasSpringBootStartStopGoals(pom) {
			t.Error("Expected HasSpringBootStartStopGoals to return true after configuration")
		}
	})

	t.Run("no spring-boot plugin", func(t *testing.T) {
		pom := &gopom.Project{}
		modified := parser.ConfigureSpringBootPlugin(pom)
		if modified {
			t.Error("Expected ConfigureSpringBootPlugin to return false for project without spring-boot plugin")
		}
	})
}
