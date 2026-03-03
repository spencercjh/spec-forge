// Package spring_test tests the Spring extractor implementation.
package spring_test

import (
	"os"
	"testing"

	"github.com/scagogogo/gradle-parser/pkg/model"
	"github.com/spencercjh/spec-forge/internal/extractor/spring"
)

func TestGradleParser_Parse(t *testing.T) {
	gradlePath := "../../../integration-tests/gradle-springboot-openapi-demo/build.gradle"

	if _, err := os.Stat(gradlePath); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	parser := spring.NewGradleParser()
	build, err := parser.Parse(gradlePath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if build == nil {
		t.Fatal("Parsed build.gradle should not be nil")
	}
}

func TestGradleParser_HasSpringdocDependency(t *testing.T) {
	gradlePath := "../../../integration-tests/gradle-springboot-openapi-demo/build.gradle"

	if _, err := os.Stat(gradlePath); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	parser := spring.NewGradleParser()
	build, err := parser.Parse(gradlePath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !parser.HasSpringdocDependency(build) {
		t.Error("HasSpringdocDependency should be true for demo project")
	}
}

func TestGradleParser_HasSpringdocPlugin(t *testing.T) {
	gradlePath := "../../../integration-tests/gradle-springboot-openapi-demo/build.gradle"

	if _, err := os.Stat(gradlePath); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	parser := spring.NewGradleParser()
	build, err := parser.Parse(gradlePath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !parser.HasSpringdocPlugin(build) {
		t.Error("HasSpringdocPlugin should be true for demo project")
	}
}

func TestGradleParser_GetSpringdocVersion(t *testing.T) {
	gradlePath := "../../../integration-tests/gradle-springboot-openapi-demo/build.gradle"

	if _, err := os.Stat(gradlePath); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	parser := spring.NewGradleParser()
	build, err := parser.Parse(gradlePath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	version := parser.GetSpringdocVersion(build)
	if version == "" {
		t.Error("Springdoc version should not be empty for demo project")
	}
}

func TestGradleParser_GetSpringBootVersion(t *testing.T) {
	gradlePath := "../../../integration-tests/gradle-springboot-openapi-demo/build.gradle"

	if _, err := os.Stat(gradlePath); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	parser := spring.NewGradleParser()
	build, err := parser.Parse(gradlePath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	version := parser.GetSpringBootVersion(build)
	if version == "" {
		t.Error("Spring Boot version should not be empty")
	}
	// The demo project uses Spring Boot 4.0.3
	if version != "4.0.3" {
		t.Logf("Warning: Expected Spring Boot 4.0.3, got %s", version)
	}
}

func TestGradleParser_FindDependency(t *testing.T) {
	parser := spring.NewGradleParser()

	// Create a minimal project for testing
	project := &model.Project{
		Dependencies: []*model.Dependency{
			{
				Group:   "org.springframework.boot",
				Name:    "spring-boot-starter-web",
				Version: "4.0.3",
				Scope:   "implementation",
			},
		},
	}

	dep := parser.FindDependency(project, "spring-boot-starter-web")
	if dep == nil {
		t.Fatal("Should find spring-boot-starter-web dependency")
	}
	if dep.Version != "4.0.3" {
		t.Errorf("Version = %s, want 4.0.3", dep.Version)
	}
}

func TestGradleParser_FindPlugin(t *testing.T) {
	parser := spring.NewGradleParser()

	// Create a minimal project for testing
	project := &model.Project{
		Plugins: []*model.Plugin{
			{
				ID:      "org.springframework.boot",
				Version: "4.0.3",
			},
		},
	}

	plugin := parser.FindPlugin(project, "org.springframework.boot")
	if plugin == nil {
		t.Fatal("Should find spring-boot plugin")
	}
	if plugin.Version != "4.0.3" {
		t.Errorf("Version = %s, want 4.0.3", plugin.Version)
	}
}
