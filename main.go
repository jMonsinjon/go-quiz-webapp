package main

import (
	"log"
	"html/template"
	"net/http"
	"github.com/gorilla/mux"
	"math/rand"
	"flag"
	"github.com/gorilla/websocket"
	"time"
	"strconv"
)

type Intro struct {
	PageTitle   string
	Video   string
	FirstQuestionURL string
}

type Question struct {
	ID       string
	PageTitle   string
	Image    string
	Response    string
	Success   bool
	WrongAnswer   bool
}

var questions []*Question
var endGameURL string

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
var tick = time.Tick(1*time.Second)
var timerStarted = false
var countdown = 3600

func main() {
	// Init vars
	initVars()

	//Initialisation des routes
	r:= mux.NewRouter()
	//Manipulation des routes
	r.HandleFunc("/", displayIntro).Methods("GET")
	r.HandleFunc("/q/{id}", getQuestion).Methods("GET")
	r.HandleFunc("/q/{id}", postAnswer).Methods("POST")
	r.HandleFunc(endGameURL, displayConclusion).Methods("GET")
	r.PathPrefix("/statics/").Handler(http.StripPrefix("/statics/", http.FileServer(http.Dir("./assets"))))
	r.HandleFunc("/timer", readTimer).Methods("GET")

	port := flag.String("p", "8080", "port to serve on")
	log.Printf("Serving on HTTP port: %s\n", *port)
	log.Fatal(http.ListenAndServe(":"+*port, r))
}

func initVars(){
	log.Print("Initializing app variables")
	endGameURL = "/" + randomString(20)
	questions =  []*Question{
		{ID: randomString(20), PageTitle: "Question 1", Image: "virus_1.jpeg", Response: "gagne"},
		{ID: randomString(20), PageTitle: "Question 2", Image: "virus_2.jpeg", Response: "gagne"},
		{ID: randomString(20), PageTitle: "Question 3", Image: "virus_3.jpeg", Response: "gagne"},
		{ID: randomString(20), PageTitle: "Question 4", Image: "virus_4.jpeg", Response: "gagne"},
	}
}

func initTimer() {
	timerStarted = true
	for countdown > 0 {
		log.Printf("Time remaining: %d", countdown)
		countdown--
		<-tick
	}
}

//func displayQuizz(w http.ResponseWriter, r *http.Request)
func displayIntro(w http.ResponseWriter, r *http.Request){
	tmpl := template.Must(template.ParseFiles("templates/intro.html"))
	data := Intro{
		PageTitle: "Bienvenue dans le quizz",
		Video: "intro.mp4",
		FirstQuestionURL: "/q/" + questions[0].ID,
	}
	tmpl.Execute(w, data)
}

func displayConclusion(w http.ResponseWriter, r *http.Request){
	tmpl := template.Must(template.ParseFiles("templates/conclusion.html"))
	data := Intro{
		PageTitle: "Terminé",
		Video: "endgame.mp4",
	}
	tmpl.Execute(w, data)
}

func getQuestion(w http.ResponseWriter, r *http.Request){
	if (!timerStarted) {
		go initTimer()
	}

	params:= mux.Vars(r)
	questionID := params["id"]
	question := getQuestionByID(questionID)

	if len(question.ID) > 0 {
		tmpl := template.Must(template.ParseFiles("templates/question.html"))
		tmpl.Execute(w, question)
	} else {
		log.Printf("The question %s doesn't exists", questionID)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func postAnswer(w http.ResponseWriter, r *http.Request){
	proposal := r.FormValue("response")
	questionID := mux.Vars(r)["id"]

	question := getQuestionByID(questionID)
	if(question.Response == proposal) {
		question.Success = true
		question.WrongAnswer = false
		next := getNextPage(question.ID)
		http.Redirect(w, r, next, http.StatusSeeOther)
	} else {
		question.Success = false
		question.WrongAnswer = true
		http.Redirect(w, r, "/q/"+question.ID, http.StatusSeeOther)
	}
}

func readTimer(w http.ResponseWriter, r *http.Request){
	ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
	}
	defer ws.Close()

	go writer(ws)
	reader(ws)
}

func reader(conn *websocket.Conn) {
    for {
    // read in a message
		_, p, err := conn.ReadMessage()
        if err != nil {
            log.Println(err)
            return
		}
		
		log.Printf("Message received from client : %s", string(p))
    }
}

func writer(conn *websocket.Conn) {
	log.Print("Start writing into the websocket")
    for {
        for countdown > 0 {
			quotient := strconv.Itoa(countdown / 60) // integer division, decimals are truncated
			remainder := strconv.Itoa(countdown % 60)
			timeRemaining := quotient + ":" + remainder

			if err := conn.WriteMessage(1, []byte(timeRemaining)); err != nil {
				log.Println(err)
				return
			}
			time.Sleep(time.Second)
		}
    }
}

func getQuestionByID(qID string)(*Question) {
	for _, q:= range questions {
		if q.ID == qID {
			return q
		}
	}
	log.Printf("The question %s doesn't exists", qID)
	return nil
}

func getNextPage(currentID string)(string){
	for i, q:= range questions {
		if q.ID == currentID {
			if(i == len(questions)-1){
				// If last question, return conclusion
				return endGameURL
			} 
			return "/q/" + questions[i+1].ID
		}
	}
	log.Printf("The question %s doesn't exists", currentID)
	return "/"
}

func randomString(n int) string {
	var letter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	b := make([]rune, n)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}