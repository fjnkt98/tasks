package settings

import (
	"os"
	"strconv"
)

var DatabaseURL = os.Getenv("DATABASE_URL")
var CorsAllowOrigin = os.Getenv("CORS_ALLOW_ORIGIN")
var GoogleCloudProjectName = os.Getenv("GOOGLE_CLOUD_PROJECT_NAME")
var OtelCollectorURL = os.Getenv("OTEL_COLLECTOR_URL")
var Port = MustParseInt(os.Getenv("PORT"))
var UseSecureCookie = ParseBool(os.Getenv("USE_SECURE_COOKIE"))

func MustParseInt(s string) int {
	port, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return port

}

func ParseBool(s string) bool {
	return s == "true"
}
