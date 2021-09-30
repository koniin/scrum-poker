// websockets.go
package main

import (
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/olahol/melody.v1"
)

// m.Broadcast([]byte("msg"))

// m.BroadcastOthers([]byte("msg"), s)

// m.BroadcastFilter([]byte("msg"),
// 	func(q *melody.Session) bool {
// 		return q.Request.URL.Path == s.Request.URL.Path
// 	})

// if r.URL.Path != "/" {
// 	http.Error(w, "Not found", http.StatusNotFound)
// 	return
// }

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "index.html")
}

type VoterInfo struct {
	ID, RoomId, UserId, Vote string
}

func (v VoterInfo) String() string {
	return v.RoomId + ":" + v.ID + ":" + v.UserId + ":" + v.Vote
}

func main() {
	voters := make(map[*melody.Session]*VoterInfo)
	lock := new(sync.Mutex)
	counter := 0
	m := melody.New()

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/voting/", func(w http.ResponseWriter, r *http.Request) {
		m.HandleRequest(w, r)
	})

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		log.Println("recieved: " + string(msg))

		p := strings.Split(string(msg), ":")
		cmd := p[0]

		lock.Lock()
		if cmd == "vote" || cmd == "reveal" || cmd == "clear" {
			voter := voters[s]

			if len(p) == 4 {
				voter.Vote = p[3]
			}

			for sess, v := range voters {
				if voter.RoomId == v.RoomId {
					sess.Write([]byte(cmd + ":" + voter.String()))
				}
			}
			log.Println(cmd + ":" + voter.String())
		}
		lock.Unlock()
	})

	m.HandleConnect(func(s *melody.Session) {
		lock.Lock()

		roomId := path.Base(s.Request.URL.Path)
		userId := s.Request.URL.Query().Get("userId")
		id := strconv.Itoa(counter)

		connectedVoter := &VoterInfo{id, roomId, userId, ""}

		// send to the one connecting what voters are already connected
		for vs, info := range voters {
			if info.RoomId == roomId {
				s.Write([]byte("joined:" + info.String()))
				vs.Write([]byte("joined:" + connectedVoter.String()))
			}
		}
		// Add a new voter to the list
		voters[s] = connectedVoter
		// Echo info to the one connecting
		s.Write([]byte("myinfo:" + voters[s].String()))

		log.Println("connected:" + voters[s].String())
		counter += 1

		lock.Unlock()
	})

	m.HandleDisconnect(func(s *melody.Session) {
		lock.Lock()
		voter := voters[s]
		for sess, v := range voters {
			if voter.RoomId == v.RoomId {
				sess.Write([]byte("disconnected:" + voter.String()))
			}
		}
		log.Println("disconnected:" + voter.String())
		delete(voters, s)
		lock.Unlock()
	})

	http.HandleFunc("/", serveHome)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s \n", port)
	}

	log.Println("Listening on port " + port)
	http.ListenAndServe(":"+port, nil)
}
