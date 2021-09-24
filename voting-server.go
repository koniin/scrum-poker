// websockets.go
package main

import (
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/olahol/melody.v1"
)

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	// if r.URL.Path != "/" {
	// 	http.Error(w, "Not found", http.StatusNotFound)
	// 	return
	// }
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
		// m.Broadcast(msg)

		// m.BroadcastFilter(msg, func(q *melody.Session) bool {
		// 	return q.Request.URL.Path == s.Request.URL.Path
		// })

		log.Println(string(msg))

		p := strings.Split(string(msg), ":")
		lock.Lock()
		if p[0] == "vote" {
			if len(p) == 4 {

				voter := voters[s]

				//socket.send("vote:" + state.id + ":" + state.userId + ":" + voteSelect.value)
				// vote:1:Göran:8
				voter.Vote = p[3]
				for sess, v := range voters {
					if voter.RoomId == v.RoomId {
						sess.Write([]byte("voted:" + voter.String()))
					}
				}

				log.Println("voted:" + voter.String())

				// m.BroadcastFilter([]byte("voted:"+voter.RoomId+":"+voter.ID+":"+voter.UserId+":"+voter.Vote),
				// 	func(q *melody.Session) bool {
				// 		return q.Request.URL.Path == s.Request.URL.Path
				// 	})
			}
		} else if p[0] == "reveal" {
			voter := voters[s]
			for sess, v := range voters {
				if voter.RoomId == v.RoomId {
					sess.Write([]byte("reveal:" + voter.String()))
				}
			}
			log.Println("reveal:" + voter.String())
		} else if p[0] == "clear" {
			voter := voters[s]
			for sess, v := range voters {
				v.Vote = ""
				if voter.RoomId == v.RoomId {
					sess.Write([]byte("clear:" + voter.String()))
				}
			}
			log.Println("clear:" + voter.String())
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

		//m.BroadcastOthers([]byte("joined:"+voters[s].RoomId+":"+voters[s].ID+":"+voters[s].UserId), s)

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

		// m.BroadcastFilter([]byte("disconnected:"+voters[s].RoomId+":"+voters[s].ID+":"+voters[s].UserId),
		// 	func(q *melody.Session) bool {
		// 		return q.Request.URL.Path == s.Request.URL.Path
		// 	})

		delete(voters, s)
		lock.Unlock()
	})

	http.HandleFunc("/", serveHome)

	log.Println("Listening on port 8080")
	http.ListenAndServe(":8080", nil)
}
