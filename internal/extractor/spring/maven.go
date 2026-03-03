package spring

import (
	"fmt"
	"strings"

	"github.com/vifraa/gopom"
)

// Springdoc group and artifact constants.
const (
	SpringdocGroupID             = "org.springdoc"
	SpringdocWebMVCArtifactID    = "springdoc-openapi-starter-webmvc-ui"
	SpringdocMavenPluginArtifact = "springdoc-openapi-maven-plugin"
	SpringBootParentGroupID      = "org.springframework.boot"
	SpringBootParentArtifactID   = "spring-boot-starter-parent"
)

// MavenParser parses and modifies Maven pom.xml files.
type MavenParser struct{}

// NewMavenParser creates a new MavenParser instance.
func NewMavenParser() *MavenParser {
	return &MavenParser{}
}

// Parse reads and parses a pom.xml file.
func (p *MavenParser) Parse(pomPath string) (*gopom.Project, error) {
	project, err := gopom.Parse(pomPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pom.xml: %w", err)
	}

	return project, nil
}

// GetSpringBootVersion extracts the Spring Boot version from a pom.
func (p *MavenParser) GetSpringBootVersion(pom *gopom.Project) string {
	// Check parent version (most common case)
	if pom.Parent != nil && pom.Parent.Version != nil {
		if pom.Parent.GroupID != nil && *pom.Parent.GroupID == SpringBootParentGroupID {
			if pom.Parent.ArtifactID != nil && *pom.Parent.ArtifactID == SpringBootParentArtifactID {
				return *pom.Parent.Version
			}
		}
	}

	// Check dependency management
	if pom.DependencyManagement != nil && pom.DependencyManagement.Dependencies != nil {
		for _, dep := range *pom.DependencyManagement.Dependencies {
			if dep.GroupID != nil && *dep.GroupID == SpringBootParentGroupID {
				if dep.ArtifactID != nil && *dep.ArtifactID == SpringBootParentArtifactID {
					if dep.Version != nil {
						return *dep.Version
					}
				}
			}
		}
	}

	return ""
}

// HasSpringdocDependency checks if the pom has springdoc dependencies.
func (p *MavenParser) HasSpringdocDependency(pom *gopom.Project) bool {
	return p.FindDependency(pom, SpringdocGroupID, SpringdocWebMVCArtifactID) != nil
}

// GetSpringdocVersion returns the springdoc version if present.
func (p *MavenParser) GetSpringdocVersion(pom *gopom.Project) string {
	dep := p.FindDependency(pom, SpringdocGroupID, SpringdocWebMVCArtifactID)
	if dep != nil && dep.Version != nil {
		// Handle ${springdoc.version} style references
		version := *dep.Version
		if strings.HasPrefix(version, "${") && strings.HasSuffix(version, "}") {
			propName := strings.Trim(version, "${}")
			if pom.Properties != nil {
				if val, ok := pom.Properties.Entries[propName]; ok {
					return val
				}
			}
		}
		return version
	}
	return ""
}

// HasSpringdocPlugin checks if the pom has the springdoc maven plugin.
func (p *MavenParser) HasSpringdocPlugin(pom *gopom.Project) bool {
	if pom.Build == nil || pom.Build.Plugins == nil {
		return false
	}

	for _, plugin := range *pom.Build.Plugins {
		if plugin.GroupID != nil && *plugin.GroupID == SpringdocGroupID {
			if plugin.ArtifactID != nil && *plugin.ArtifactID == SpringdocMavenPluginArtifact {
				return true
			}
		}
	}

	return false
}

// FindDependency finds a dependency by groupId and artifactId.
func (p *MavenParser) FindDependency(pom *gopom.Project, groupID, artifactID string) *gopom.Dependency {
	if pom.Dependencies == nil {
		return nil
	}

	for i := range *pom.Dependencies {
		dep := &(*pom.Dependencies)[i]
		if dep.GroupID != nil && *dep.GroupID == groupID {
			if dep.ArtifactID != nil && *dep.ArtifactID == artifactID {
				return dep
			}
		}
	}

	return nil
}

// AddDependency adds a dependency to the pom.
func (p *MavenParser) AddDependency(pom *gopom.Project, groupID, artifactID, version string) {
	dep := gopom.Dependency{
		GroupID:    &groupID,
		ArtifactID: &artifactID,
		Version:    &version,
	}

	if pom.Dependencies == nil {
		pom.Dependencies = &[]gopom.Dependency{dep}
	} else {
		*pom.Dependencies = append(*pom.Dependencies, dep)
	}
}

// AddPlugin adds a plugin to the pom build section.
func (p *MavenParser) AddPlugin(pom *gopom.Project, groupID, artifactID, version string) {
	goals := []string{"generate"}
	plugin := gopom.Plugin{
		GroupID:    &groupID,
		ArtifactID: &artifactID,
		Version:    &version,
		Executions: &[]gopom.PluginExecution{
			{
				Goals: &goals,
			},
		},
	}

	if pom.Build == nil {
		pom.Build = &gopom.Build{}
	}
	if pom.Build.Plugins == nil {
		pom.Build.Plugins = &[]gopom.Plugin{plugin}
	} else {
		*pom.Build.Plugins = append(*pom.Build.Plugins, plugin)
	}
}

// GetModules returns the list of modules in a multi-module Maven project.
func (p *MavenParser) GetModules(pom *gopom.Project) []string {
	if pom.Modules == nil {
		return nil
	}

	var modules []string
	for _, module := range *pom.Modules {
		if module != "" {
			modules = append(modules, module)
		}
	}
	return modules
}

// HasSpringBootPlugin checks if the pom has the Spring Boot Maven plugin.
func (p *MavenParser) HasSpringBootPlugin(pom *gopom.Project) bool {
	if pom.Build == nil || pom.Build.Plugins == nil {
		return false
	}

	for _, plugin := range *pom.Build.Plugins {
		if plugin.GroupID != nil && *plugin.GroupID == SpringBootParentGroupID {
			if plugin.ArtifactID != nil && *plugin.ArtifactID == "spring-boot-maven-plugin" {
				return true
			}
		}
	}

	return false
}
