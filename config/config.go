package config

import (
	"log"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Config 结构体用于存储应用程序的配置信息
type Config struct {
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	JWTSecret          string
	LogLevel           string
	SMTPHost           string
	SMTPPort           int
	SMTPUsername       string
	SMTPPassword       string
	DomainName         string
	FrontendURL        string
	BackendURL         string
	S3Region           string
	S3Bucket           string
	GCSProjectID       string
	GCSBucketName      string
	GCSCredentialsFile string
	LocalStoragePath   string
	Debug              bool // 是否开启调试模式
}

// AppConfig 是全局配置变量
var AppConfig Config

// Init 函数用于初始化配置
func Init() {
	// 加载 .env 文件
	err := godotenv.Load()
	if err != nil {
		log.Printf("警告：无法加载 .env 文件: %v", err)
	}

	// 从环境变量中读取配置
	AppConfig = Config{
		DBHost:             getEnv("DB_HOST", ""),
		DBPort:             getEnv("DB_PORT", ""),
		DBUser:             getEnv("DB_USER", ""),
		DBPassword:         getEnv("DB_PASSWORD", ""),
		DBName:             getEnv("DB_NAME", ""),
		JWTSecret:          getEnv("JWT_SECRET", ""),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		SMTPHost:           getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:           getEnvAsInt("SMTP_PORT", 465),
		SMTPUsername:       getEnv("SMTP_USERNAME", ""),
		SMTPPassword:       getEnv("SMTP_PASSWORD", ""),
		DomainName:         getEnv("DOMAIN_NAME", "localhost"),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:5173"),
		BackendURL:         getEnv("BACKEND_URL", "http://localhost:8080"),
		S3Region:           getEnv("S3_REGION", "us-west-2"),
		S3Bucket:           getEnv("S3_BUCKET", "your-bucket-name"),
		GCSProjectID:       getEnv("GCS_PROJECT_ID", ""),
		GCSBucketName:      getEnv("GCS_BUCKET_NAME", ""),
		GCSCredentialsFile: getEnv("GCS_CREDENTIALS_FILE", ""),
		LocalStoragePath:   getEnv("LOCAL_STORAGE_PATH", "./uploads"),
		Debug:              getEnvAsBool("DEBUG", true),
	}

	// 在 Init 函数中临时修改日志级别
	AppConfig.LogLevel = "debug"

	validateConfig()

	// 如果是调试模式，打印更详细的路由信息
	if AppConfig.Debug {
		gin.SetMode(gin.DebugMode)
		log.Println("应用程序运行在调试模式")
	} else {
		gin.SetMode(gin.ReleaseMode)
		log.Println("应用程序运行在生产模式")
	}

	log.Printf("配置加载完成。数据库：%s:%s", AppConfig.DBHost, AppConfig.DBPort)
	log.Printf("SMTP配置：主机=%s，端口=%d，用户名=%s", AppConfig.SMTPHost, AppConfig.SMTPPort, AppConfig.SMTPUsername)
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultVal int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}

func getEnvAsBool(key string, defaultVal bool) bool {
	valStr := getEnv(key, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}
	return defaultVal
}

func validateConfig() {
	if AppConfig.DBHost == "" || AppConfig.DBPort == "" || AppConfig.DBUser == "" || AppConfig.DBPassword == "" || AppConfig.DBName == "" {
		log.Fatal("错误：数据库配置不完整")
	}
	if AppConfig.JWTSecret == "" {
		log.Fatal("错误：JWT密钥未设置")
	}
	if AppConfig.SMTPHost == "" || AppConfig.SMTPUsername == "" || AppConfig.SMTPPassword == "" {
		log.Fatal("错误：SMTP配置不完整")
	}
}
