# Spring Boot

Spec Forge uses [springdoc-openapi](https://springdoc.org/) to generate OpenAPI specs from Spring Boot projects.

## How It Works

1. **Detection**: Automatically detects Maven or Gradle build files
2. **Patching**: Injects the springdoc-openapi plugin and configures the Spring Boot plugin
3. **Generation**: Runs the appropriate build command to generate the OpenAPI spec

## Supported Build Tools

| Build Tool | Command Used |
|------------|--------------|
| Maven | `mvn verify` |
| Gradle | `gradle generateOpenApiDocs` |

## Multi-module Projects

For Maven multi-module projects, spec-forge automatically configures the `spring-boot-maven-plugin` with start/stop goals to handle module dependencies.

## Usage

```bash
# Basic generation
spec-forge generate ./my-spring-boot-project

# With AI enrichment
LLM_API_KEY="your-key" spec-forge generate ./my-spring-boot-project --enrich --language zh
```

## References

- [springdoc-openapi Documentation](https://springdoc.org/#plugins)
