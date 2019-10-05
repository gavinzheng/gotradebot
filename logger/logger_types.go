package logger

import (
	"io"
	"log"
	"os"
)

// Logging struct that holds all user configurable options for the logger
type Logging struct {
	Enabled              *bool  `json:"enabled,omitempty"`
	File                 string `json:"file"`
	ColourOutput         bool   `json:"colour"`
	ColourOutputOverride bool   `json:"colourOverride,omitempty"`
	Level                string `json:"level"`
	Rotate               bool   `json:"rotate"`
}

var (
	debugLogger *log.Logger
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	fatalLogger *log.Logger

	logFileHandle *os.File

	logOutput io.Writer

	// LogPath location to store logs in
	LogPath string

	// Logger create a pointer to Logging struct for holding data
	Logger = &Logging{}
)
