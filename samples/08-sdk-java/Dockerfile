FROM maven:3.9-eclipse-temurin-21 AS builder
WORKDIR /build
COPY sdks/java/ /build/sdk/
WORKDIR /build/sdk
RUN mvn install -q -DskipTests
COPY samples/08-sdk-java/pom.xml /build/app/pom.xml
COPY samples/08-sdk-java/src/ /build/app/src/
WORKDIR /build/app
RUN mvn package -q -DskipTests

FROM eclipse-temurin:21-jre-alpine
COPY --from=builder /build/app/target/product-service-1.0.0.jar /app.jar
EXPOSE 3000
CMD ["java", "-jar", "/app.jar"]
