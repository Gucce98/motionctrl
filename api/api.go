package api

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kpango/glg"

	"../config"
	"../motion"
	"../utils"
	"../version"
)

var handlersMap = map[string]func(*gin.Context){
	"/control/startup":  startHandler,
	"/control/shutdown": stopHandler,
	"/control/status":   statusHandler,
	"/control/restart":  restartHandler,
	"/detection/status": isMotionDetectionEnabled,
	"/detection/start":  startDetectionHandler,
	"/detection/pause":  pauseDetectionHandler,
	"/camera":           proxyStream,
	"/config/list":      listConfigHandler,
	"/config/set":       setConfigHandler,
	"/config/get":       getConfigHandler,
	"/config/write":     writeConfigHandler,
}

func Init() {
	glg.Info("Initializing REST API ...")
	var group *gin.RouterGroup
	r := gin.Default()

	if config.Get().Username != "" && config.Get().Password != "" {
		glg.Info("Username and password defined, authentication enabled")
		group = r.Group("/", gin.BasicAuth(gin.Accounts{config.Get().Username: config.Get().Password}), needMotionUp)
	} else {
		glg.Warn("Username and password not defined, authentication disabled")
		group = r.Group("/", needMotionUp)
	}

	for k, v := range handlersMap {
		group.GET(k, v)
	}

	r.Run(fmt.Sprintf("%s:%d", config.Get().Address, config.Get().Port))
}

func needMotionUp(c *gin.Context) {

	/** Every request, except for /control* requests, need motion up and running**/

	if !strings.HasPrefix(fmt.Sprint(c.Request.URL), "/control") {
		motionStarted := motion.IsStarted()

		if !motionStarted {
			c.JSON(http.StatusConflict, gin.H{"message": "motion was not started yet"})
			return
		}
	}
}

func startHandler(c *gin.Context) {
	motionDetection, err := strconv.ParseBool(c.DefaultQuery("detection", "false"))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "'detection' parameter must be 'true' or 'false'"})
	} else {
		err = motion.Startup(motionDetection)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		} else {
			c.JSON(http.StatusOK, gin.H{"message": "motion started"})
		}
	}

}

func restartHandler(c *gin.Context) {
	err := motion.Restart()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "motion restarted"})
	}
}

func stopHandler(c *gin.Context) {
	err := motion.Shutdown()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "motion stopped"})
	}
}

func statusHandler(c *gin.Context) {
	started := motion.IsStarted()

	c.JSON(http.StatusOK, gin.H{"motionStarted": started})
}

func isMotionDetectionEnabled(c *gin.Context) {
	enabled, err := motion.IsMotionDetectionEnabled()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
	} else {
		c.JSON(http.StatusOK, gin.H{"motionDetectionEnabled": enabled})
	}
}

func startDetectionHandler(c *gin.Context) {
	err := motion.EnableMotionDetection(true)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "motion detection started"})
	}
}

func pauseDetectionHandler(c *gin.Context) {
	err := motion.EnableMotionDetection(false)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "motion detection paused"})
	}
}

//proxyStream is a courtesy of: https://github.com/gin-gonic/gin/issues/686
func proxyStream(c *gin.Context) {
	url, _ := url.Parse(motion.GetStreamBaseURL())
	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.ServeHTTP(c.Writer, c.Request)
}

func listConfigHandler(c *gin.Context) {
	configMap, err := motion.ConfigList()
	glg.Log(c.Request.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
	} else {
		c.JSON(http.StatusOK, configMap)
	}
}

func getConfigHandler(c *gin.Context) {
	query := c.Query("query")

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "'query' parameter not specified"})
	} else {
		config, err := motion.ConfigGet(query)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}) //TODO improve fail with returned status code from request sent to motion
		} else {
			c.JSON(http.StatusOK, config)
		}
	}
}

func setConfigHandler(c *gin.Context) {
	writeback, err := strconv.ParseBool(c.DefaultQuery("writeback", "false"))

	if err != nil {
		nameAndValue := utils.RegexSubmatchTypedMap("/config/set\\?("+motion.KeyValueRegex+"+)=("+motion.KeyValueRegex+"+)", fmt.Sprint(c.Request.URL), motion.ReverseConfigTypeMapper)

		if len(nameAndValue) != 1 {
			c.JSON(http.StatusBadRequest, gin.H{"message": "'name' and 'value' parameters not specified"})
		} else {
			for k, v := range nameAndValue {
				b := motion.ConfigCanSet(k)
				if b {
					err = motion.ConfigSet(k, v.(string))
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}) //TODO improve fail with returned status code from request sent to motion
					} else {

						if writeback {
							err = motion.ConfigWrite()
							if err != nil {
								c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
								return
							}
						}

						c.JSON(http.StatusOK, gin.H{k: motion.ConfigTypeMapper(v.(string))})

					}
				} else {
					c.JSON(http.StatusForbidden, gin.H{"message": fmt.Sprintf("'%s' cannot be updated with %s", k, version.Name)})
				}

			}

		}
	} else {
		c.JSON(http.StatusBadGateway, gin.H{"message": "'writeback' parameter must be true/false"})
	}

}

func writeConfigHandler(c *gin.Context) {
	err := motion.ConfigWrite()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "configuration written to file"})
	}
}
