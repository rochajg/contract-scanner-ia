package main

import (
	_ "embed"
	"log"
	"os"

	"contract-scanner/cmd/contract-scanner-api/config"
	"contract-scanner/internal/handler"
	postgres "contract-scanner/internal/infra/database/postgres"
	"contract-scanner/internal/infra/database/postgres/repository"
	"contract-scanner/internal/infra/auth"
	"contract-scanner/internal/infra/llm"
	"contract-scanner/internal/infra/pdf/pdfpipeline"
	"contract-scanner/internal/infra/storage"
	"contract-scanner/internal/usecase"

	"github.com/joho/godotenv"
)

//go:embed prompt.md
var systemPrompt string

func main() {
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")
	_ = godotenv.Load("../../../.env")

	// clerk.SetKey(os.Getenv("CLERK_SECRET_KEY")) // disabled for local testing

	dbClient := postgres.NewClient(postgres.DatabaseConfig{
		Host:     os.Getenv("DB_HOST"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Name:     os.Getenv("DB_NAME"),
		Port:     os.Getenv("DB_PORT"),
		SslMode:  os.Getenv("DB_SSLMODE"),
	})

	db, err := dbClient.Open()
	if err != nil {
		log.Fatal("error connecting to database: ", err)
	}

	if err := dbClient.Migrate(db); err != nil {
		log.Fatal("error running migrations: ", err)
	}

	log.Println("database connected and migrated")

	s3Client := storage.NewS3Client(
		os.Getenv("AWS_BUCKET"),
		os.Getenv("AWS_REGION"),
		os.Getenv("ACCESS_KEY"),
		os.Getenv("SECRET_ACCESS_KEY"),
	)

	analyseRepo := repository.NewAnalyseRepo(db)
	userRepo := repository.NewUserRepo(db)

	jwtSvc := auth.NewJWTService()

	pdfExtractor := pdfpipeline.NewPipelineExtractor()
	openaiClient := llm.NewOpenAIClient(llm.ClientConfig{
		APIKey:       os.Getenv("LLM_API_KEY"),
		BaseURL:      os.Getenv("LLM_BASE_URL"),
		Model:        os.Getenv("LLM_MODEL"),
		SystemPrompt: systemPrompt,
	})

	uploadPDF := usecase.NewUploadPDF(analyseRepo, s3Client)
	processContract := usecase.NewProcessContract(analyseRepo, s3Client, pdfExtractor, openaiClient)
	listAnalyses := usecase.NewListAnalyses(analyseRepo)
	deleteAnalysis := usecase.NewDeleteAnalysis(analyseRepo, s3Client)
	registerUser := usecase.NewRegisterUser(userRepo, jwtSvc)
	loginUser := usecase.NewLoginUser(userRepo, jwtSvc)

	uploadHandler := handler.NewUploadHandler(uploadPDF)
	analyseHandler := handler.NewAnalyseHandler(processContract, listAnalyses, deleteAnalysis)
	authHandler := handler.NewAuthHandler(registerUser, loginUser)

	r := config.Routes(uploadHandler, analyseHandler, authHandler, jwtSvc)

	log.Println("server running on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
