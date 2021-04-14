package server

import (
	"github.com/Reeceeboii/personal-website-backend/pkg/email"
	"github.com/Reeceeboii/personal-website-backend/pkg/logging"

	// my packages
	"github.com/Reeceeboii/personal-website-backend/pkg/middleware"

	// AWS
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	// Mux
	"github.com/gorilla/mux"

	"log"
	"net/http"
	"os"
	"runtime"
	"time"
)

var BackendServer *Server

// Holds various pieces of non-changing data about the current main release
type StaticInformation struct {
	// current Go runtime version
	GoRuntime string
	// name of the current AWS SDK
	AWSSDKName string
	// current AWS SDK version
	AWSSDKVersion string
	// the time that the main was created
	ServerBootTime time.Time
}

// Server struct
type Server struct {
	// instance of StaticInformation
	StaticInformation StaticInformation
	// HTTP client used to make outbound HTTP requests to external APIs (i.e. GitHub)
	HTTPClient http.Client
	// AWS SDK session
	AWSSession *session.Session
	// AWS S3-specific session
	S3Session *s3.S3
	// Registers API routers and manages dispatching handlers
	Router *mux.Router
	// Manages outgoing emails and Gmail authentication
	EmailManager *email.Manager
	// Handle logging to stdout
	Logger *logging.Logger
}

// carry out operations at the start of the main's lifetime
func (s *Server) serverStartupOperations() {
	s.StaticInformation.GoRuntime = runtime.Version()
	s.StaticInformation.AWSSDKName = aws.SDKName
	s.StaticInformation.AWSSDKVersion = aws.SDKVersion
	s.StaticInformation.ServerBootTime = time.Now()

	// send out the server boot email
	s.EmailManager.SendServerStartupEmail(
		s.StaticInformation.ServerBootTime,
		s.StaticInformation.GoRuntime,
		s.StaticInformation.AWSSDKName,
		s.StaticInformation.AWSSDKVersion)
}

// Create a new Server
func NewServer() *Server {
	// create a new AWS session
	AWSSession, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})
	if err != nil {
		log.Fatalf("Error creating new AWS session %s", err.Error())
	}

	// create new Mux router
	router := mux.NewRouter().StrictSlash(true)
	// apply logging, CORS and caching middleware
	router.Use(middleware.LoggingMiddleware)
	router.Use(middleware.CORSMiddleware)

	// router.Use(middleware.CacheMiddleware)

	router.HandleFunc("/", Root).Methods(http.MethodGet)

	// register photography (AWS S3) routes
	router.HandleFunc("/api/photos/list-collections", ListCollections).Methods(http.MethodGet)
	router.HandleFunc("/api/photos/get-contents", GetCollectionContents).Methods(http.MethodGet)
	// register routes for GitHub data

	// for any non matching routes, send a 404 response back
	router.NotFoundHandler = http.HandlerFunc(FourOhFour)

	serv := Server{
		StaticInformation: StaticInformation{
			GoRuntime:      runtime.Version(),
			AWSSDKName:     aws.SDKName,
			AWSSDKVersion:  aws.SDKVersion,
			ServerBootTime: time.Now(),
		},
		HTTPClient: http.Client{
			// 10 second timeout stops silent failures if any external endpoints never respond
			Timeout: time.Second * 10,
		},
		AWSSession: AWSSession,
		// create a new S3 specific session
		S3Session: s3.New(AWSSession),
		Router:    router,
		// create a new email manager
		EmailManager: email.NewEmailManager(),
		// create a new logger
		Logger: logging.NewLogger(),
	}

	// run startup operations and return the new main
	serv.serverStartupOperations()
	return &serv
}
