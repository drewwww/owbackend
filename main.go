// Use of this source code is governed by the GPLv3
// that can be found in the COPYING file.

package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/TF2Stadium/Helen/config"
	"github.com/TF2Stadium/Helen/controllers"
	chelpers "github.com/TF2Stadium/Helen/controllers/controllerhelpers"
	"github.com/TF2Stadium/Helen/controllers/socket"
	"github.com/TF2Stadium/Helen/database"
	"github.com/TF2Stadium/Helen/database/migrations"
	"github.com/TF2Stadium/Helen/helpers"
	_ "github.com/TF2Stadium/Helen/helpers/authority" // to register authority types
	_ "github.com/TF2Stadium/Helen/internal/pprof"    // to setup expvars
	"github.com/TF2Stadium/Helen/internal/version"
	"github.com/TF2Stadium/Helen/models/chat"
	"github.com/TF2Stadium/Helen/models/event"
	"github.com/TF2Stadium/Helen/models/lobby"
	"github.com/TF2Stadium/Helen/models/lobby_settings"
	"github.com/TF2Stadium/Helen/models/rpc"
	"github.com/TF2Stadium/Helen/routes"
	socketServer "github.com/TF2Stadium/Helen/routes/socket"
	"github.com/rs/cors"
)

var (
	flagGen = flag.Bool("genkey", false, "write a 32bit key for encrypting cookies the given file, and exit")
	docPrint = flag.Bool("printdoc", false, "print the docs for environment variables, and exit.")
	dbMaxopen = flag.Int("db-maxopen", 80, "maximum number of open database connections allowed.")
)

func main() {
	flag.Parse()

	if *flagGen {
		key := make([]byte, 64)
		_, err := rand.Read(key)
		if err != nil {
			logrus.Fatal(err)
		}

		base64Key := base64.StdEncoding.EncodeToString(key)
		fmt.Println(base64Key)
		return
	}
	if *docPrint {
		config.PrintConfigDoc()
		os.Exit(0)
	}

	logrus.Debug("Commit: ", version.GitCommit)
	logrus.Debug("Branch: ", version.GitBranch)
	logrus.Debug("Build date: ", version.BuildDate)

	controllers.InitTemplates()
	//models.ReadServers()

	err := os.Mkdir(config.Constants.DemosFolder, 0755)
	if err != nil && !os.IsExist(err) {
		logrus.Fatal(err)
	}

	if config.Constants.ProfilerAddr != "" {
		go http.ListenAndServe(config.Constants.ProfilerAddr, nil)
		logrus.Info("Running Profiler at ", config.Constants.ProfilerAddr)
	}

	database.Init()
	database.DB.DB().SetMaxOpenConns(*dbMaxopen)
	migrations.Do()

	helpers.ConnectAMQP()
	event.StartListening()
	helpers.InitGeoIPDB()

	err = lobbySettings.LoadLobbySettingsFromFile("assets/lobbySettingsData.json")
	if err != nil {
		logrus.Fatal(err)
	}

	lobby.CreateLocks()
	rpc.ConnectRPC(helpers.AMQPConn)
	lobby.RestoreServemeChecks()
	//go models.TFTVStreamStatusUpdater()

	if config.Constants.SteamIDWhitelist != "" {
		go chelpers.WhitelistListener()
	}

	httpMux := http.NewServeMux()
	//httpsMux := http.NewServeMux()
	routes.SetupHTTP(httpMux)
	//routes.SetupHTTPS(httpsMux)
	socket.RegisterHandlers()

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   config.Constants.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowCredentials: true,
	}).Handler(httpMux)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		<-sig
		shutdown()
		os.Exit(0)
	}()

	logrus.Info("Serving on ", config.Constants.ListenAddress)
	logrus.Info("Hosting on ", config.Constants.PublicAddress)
	logrus.Info("Steam web API: ", config.Constants.SteamDevAPIKey)
	go http.ListenAndServe(config.Constants.ListenAddressHTTP, http.HandlerFunc(redirectToHTTPS))
	logrus.Fatal(http.ListenAndServeTLS(config.Constants.ListenAddress, "server.pem", "server.key", corsHandler))
	//logrus.Fatal(http.ListenAndServe(config.Constants.ListenAddress, corsHandler))
}

//fucking golang being retarded
func redirectToHTTPS(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, config.Constants.ListenAddress, http.StatusMovedPermanently)
}


func shutdown() {
	logrus.Info("Received SIGINT/SIGTERM")
	chat.SendNotification(`Backend will be going down for a while for an update, click on "Reconnect" to reconnect to TF2Stadium`, 0)
	logrus.Info("waiting for GlobalWait")
	helpers.GlobalWait.Wait()
	logrus.Info("waiting for socket requests to complete.")
	socketServer.Wait()
	logrus.Info("closing all active websocket connections")
	socketServer.AuthServer.Close()
	logrus.Info("stopping event listener")
	event.StopListening()
}
