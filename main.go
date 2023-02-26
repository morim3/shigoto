package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// album represents data about a record album.
type job struct {
	ID      int `json:"id"`
	State   string `json:"state"`
	Output  string `json:"output"`
	Command string `json:"command"`
}
type postjob struct {
	Output  string `json:"output"`
	Command string `json:"command"`
}

type statenotify struct {
	ID int `json:"id"`
	State string `json:"state"`
}

var queue = []job{}

var queue_head int = 0


func main() {
	router := gin.Default()

	router.GET("/getjob", getJob)
	router.GET("/getqueue", getQueue)
	router.POST("/notifystate", notifyState)
	router.POST("/postjob", postJob)

	router.Run("localhost:8080")
}


func getJob(c *gin.Context) {
	if len(queue) > queue_head {
    c.IndentedJSON(http.StatusOK, queue[queue_head])
    queue[queue_head].State = "running"
    queue_head = queue_head + 1
	}else{
    c.IndentedJSON(http.StatusOK, "{}")
  }
}

func getQueue(c *gin.Context) {
  c.IndentedJSON(http.StatusOK, queue)
}

// postAlbums adds an album from JSON received in the request body.
func postJob(c *gin.Context) {
	var postJob postjob

	// Call BindJSON to bind the received JSON to
	// newAlbum.
	if err := c.BindJSON(&postJob); err != nil {
		return
	}

  newJob := job{ID: len(queue), State: "queued", Output: postJob.Output, Command: postJob.Command}  

	// Add the new album to the slice.
	queue = append(queue, newJob)
	c.IndentedJSON(http.StatusCreated, newJob)
}

func notifyState(c *gin.Context) {
	var stateNotification statenotify

	// Call BindJSON to bind the received JSON to
	// newAlbum.
	if err := c.BindJSON(&stateNotification); err != nil {
		return
	}

  queue[stateNotification.ID].State = stateNotification.State
	c.IndentedJSON(http.StatusCreated, queue)
}


// // getAlbumByID locates the album whose ID value matches the id
// // parameter sent by the client, then returns that album as a response.
// func getAlbumByID(c *gin.Context) {
// 	id := c.Param("id")
//
// 	// Loop through the list of albums, looking for
// 	// an album whose ID value matches the parameter.
// 	for _, a := range albums {
// 		if a.ID == id {
// 			c.IndentedJSON(http.StatusOK, a)
// 			return
// 		}
// 	}
// 	c.IndentedJSON(http.StatusNotFound, gin.H{"message": "album not found"})
// }
