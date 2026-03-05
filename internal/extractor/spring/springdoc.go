// Package spring provides Spring framework specific extraction functionality.
package spring

import "github.com/spencercjh/spec-forge/internal/extractor"

const (
	// BuildToolMaven represents Maven build tool.
	BuildToolMaven extractor.BuildTool = "maven"
	// BuildToolGradle represents Gradle build tool.
	BuildToolGradle extractor.BuildTool = "gradle"
)

// Default version constants (convention over configuration).
const (
	DefaultSpringdocVersion      = "3.0.2"
	DefaultSpringdocMavenPlugin  = "1.5"
	DefaultSpringdocGradlePlugin = "1.9.0"
)

// Springdoc constants used across Maven and Gradle parsers.
const (
	SpringdocGroupID             = "org.springdoc"
	SpringdocWebMVCArtifactID    = "springdoc-openapi-starter-webmvc-ui"
	SpringdocMavenPluginArtifact = "springdoc-openapi-maven-plugin"
	SpringdocGradlePluginID      = "org.springdoc.openapi-gradle-plugin"
)

// Spring Boot constants.
const (
	SpringBootParentGroupID    = "org.springframework.boot"
	SpringBootParentArtifactID = "spring-boot-starter-parent"
)
