// Package spring_test tests the Spring extractor implementation.
package spring_test

import (
	"os"
	"testing"

	"github.com/vifraa/gopom"

	"github.com/spencercjh/spec-forge/internal/extractor/spring"
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

	parser := spring.NewMavenParser()
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

	parser := spring.NewMavenParser()
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

	parser := spring.NewMavenParser()
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

	parser := spring.NewMavenParser()
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

	parser := spring.NewMavenParser()
	pom, err := parser.Parse(pomPath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !parser.HasSpringdocPlugin(pom) {
		t.Error("HasSpringdocPlugin should be true for demo project")
	}
}

func TestMavenParser_FindDependency(t *testing.T) {
	parser := spring.NewMavenParser()

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
