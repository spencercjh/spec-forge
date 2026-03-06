// Package spring provides Spring framework specific extraction functionality.
package spring

// Info contains Spring Boot framework specific information.
type Info struct {
	SpringBootVersion  string   // Spring Boot version
	HasSpringdocDeps   bool     // Whether springdoc dependencies exist
	HasSpringdocPlugin bool     // Whether springdoc plugin is configured
	SpringdocVersion   string   // Existing springdoc version if any
	IsMultiModule      bool     // Whether this is a multi-module project
	Modules            []string // List of module names
	MainModule         string   // The main application module
	MainModulePath     string   // Path to the main module's build file
}

const (
	FrameworkSpringBoot = "springboot"
)
