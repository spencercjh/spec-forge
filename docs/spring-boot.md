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

**Wrapper Priority:** spec-forge prefers project wrappers over system binaries:
1. `./mvnw` / `./gradlew` in project root
2. Wrapper in parent directory (for multi-module projects)
3. System `mvn` / `gradle` (fallback)

## Multi-module Projects

For Maven multi-module projects, spec-forge automatically configures the `spring-boot-maven-plugin` with start/stop goals to handle module dependencies.

## Usage

```bash
# Basic generation
spec-forge generate ./my-spring-boot-project

# With AI enrichment
spec-forge generate ./my-spring-boot-project --language zh
```

## References

- [springdoc-openapi Documentation](https://springdoc.org/#plugins)
