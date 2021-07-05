package httpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aler9/rtsp-simple-server/internal/logger"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"sync"
)

type Server struct {
	l         *http.Server
	parent    Parent
	wg        sync.WaitGroup
	ctx       context.Context
	ctxCancel func()
}

// Log is the main logging function.
func (s *Server) Log(level logger.Level, format string, args ...interface{}) {
	s.parent.Log(level, "[HTTP] "+format, append([]interface{}{}, args...)...)
}

// Parent is implemented by program.
type Parent interface {
	Log(logger.Level, string, ...interface{})
}

// New allocates a Server.
func New(
	address string,
	parent Parent,
	ctxParent context.Context,

) (*Server, error) {
	ctx, ctxCancel := context.WithCancel(ctxParent)
	s := &Server{
		l: &http.Server{
			Addr: address,
		},
		ctx:       ctx,
		ctxCancel: ctxCancel,
		parent:    parent,
	}

	s.Log(logger.Info, "listener opened on %s", address)
	s.wg.Add(1)

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/stream", streamHandler)

	go func() {
		defer s.wg.Done()

		err := s.l.ListenAndServe()
		if err != nil {
			println("ERROR")
			return
		}
	}()

	return s, nil
}

func (s *Server) Close(){
	err := s.l.Shutdown(s.ctx)
	if err != nil {
		return
	}
	s.wg.Wait()

}

// Handler part

type Req struct {
	Id string
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Methods", "DELETE, POST, GET, OPTIONS")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
		return
	}

	http.Error(w, "Running", http.StatusOK)
}

func streamHandler(w http.ResponseWriter, r *http.Request) {
	var request Req
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Methods", "DELETE, POST, GET, OPTIONS")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.URL.Path != "/stream" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
		return
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad json request must contain Id field", http.StatusNotFound)
		return
	}

	err = json.Unmarshal([]byte(b), &request)
	if err != nil {
		http.Error(w, "Bad json request must contain Id field", http.StatusNotFound)
		return
	}

	if len(request.Id) == 0 {
		http.Error(w, "Bad json request must contain Id field", http.StatusNotFound)
		return
	}

	cmd := exec.Command("sh", "-c", `youtube-dl -f best -g https://www.youtube.com/watch\?v\=`+request.Id)

	pipe, err := cmd.StdoutPipe()

	if err != nil {
		log.Fatal(err.Error())
	}

	if err := cmd.Start(); err != nil {
		// handle error
	}

	reader := bufio.NewReader(pipe)

	url, err := reader.ReadString('\n')

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(url)
	fmt.Println("ffmpeg -i '" + url + "' -vcodec libx264 -f rtsp -rtsp_transport tcp rtsp://localhost:8554/mystream")

	cmd2 := exec.Command("sh", "-c", "ffmpeg -i '"+url+"' -vcodec libx264 -f rtsp -rtsp_transport tcp rtsp://localhost:8554/mystream")

	go func() {
		err := cmd2.Start()
		if err != nil {
			log.Fatal(err)
		}

		err = cmd2.Wait()
		if err != nil {
			log.Fatal(err)
			return
		}
	}()

	http.Error(w, "Ok", http.StatusOK)

}
