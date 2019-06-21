package main

import (
	"flag"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Home struct {
	ButtonText string
	IntroURL   string
}

type Intro struct {
	Video            string
	FirstQuestionURL string
}

type Conclusion struct {
	Video string
}

type Question struct {
	ID             string
	Image          string
	Legend         string
	Response       string
	EndGameDeadURL string
	Success        bool
	WrongAnswer    bool
}

var questions []*Question
var introURL string
var endGameURL string
var endGameDeadURL string

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
var tick = time.Tick(1 * time.Second)
var timerStarted = false
var countdown int
var stopTimer = make(chan struct{})

func main() {
	// Init vars
	initVars()

	//Initialisation des routes
	r := mux.NewRouter()
	//Manipulation des routes
	r.HandleFunc("/", displayHome).Methods("GET")
	r.HandleFunc(introURL, displayIntro).Methods("GET")
	r.HandleFunc("/q/{id}", getQuestion).Methods("GET")
	r.HandleFunc("/q/{id}", postAnswer).Methods("POST")
	r.HandleFunc(endGameURL, displayConclusion).Methods("GET")
	r.HandleFunc(endGameDeadURL, displayEndDead).Methods("GET")
	r.PathPrefix("/statics/").Handler(http.StripPrefix("/statics/", http.FileServer(http.Dir("./assets"))))
	r.HandleFunc("/timer", readTimer).Methods("GET")

	port := flag.String("p", "8080", "port to serve on")
	log.Printf("Serving on HTTP port: %s\n", *port)
	log.Fatal(http.ListenAndServe(":"+*port, r))
}

func initVars() {
	log.Print("Initializing app variables")
	introURL = "/" + randomString(20)
	endGameURL = "/" + randomString(20)
	endGameDeadURL = "/" + randomString(20)
	questions = []*Question{
		{ID: randomString(20), Legend: "Bacille de Koch", Image: "bacille-de-koch.jpg", EndGameDeadURL: endGameDeadURL, Response: "TUBERCULOSE"},
		{ID: randomString(20), Legend: "Caryotype masculin", Image: "caryotype-1.jpg", EndGameDeadURL: endGameDeadURL, Response: "SYNDROME DE DOWN"},
		{ID: randomString(20), Legend: "Clostridium tetanii", Image: "clostridium-tetanii.png", EndGameDeadURL: endGameDeadURL, Response: "TETANOS"},
		{ID: randomString(20), Legend: "Caryotype féminin", Image: "caryotype-2.jpg", EndGameDeadURL: endGameDeadURL, Response: "MONOSOMIE 7"},
		{ID: randomString(20), Legend: "Virus MV", Image: "virus-mv.jpg", EndGameDeadURL: endGameDeadURL, Response: "ROUGEOLE"},
		{ID: randomString(20), Legend: "Caryotype masculin", Image: "caryotype-3.png", EndGameDeadURL: endGameDeadURL, Response: "SYNDROME DE TURNER"},
		{ID: randomString(20), Legend: "Treponema palidium", Image: "treponema.jpg", EndGameDeadURL: endGameDeadURL, Response: "SYPHILIS"},
		{ID: randomString(20), Legend: "Zaïre ebolavirus", Image: "zaire-ebolavirus.jpg", EndGameDeadURL: endGameDeadURL, Response: "EBOLA"},
		{ID: randomString(20), Legend: "Sarcopte", Image: "sarcopte.jpg", EndGameDeadURL: endGameDeadURL, Response: "GALE"},
		{ID: randomString(20), Legend: "Caryotype masculin", Image: "caryotype-4.png", EndGameDeadURL: endGameDeadURL, Response: "SYNDROME DE KLINEFELTER"},
	}
}

func initTimer() {
	countdown = 3000
	timerStarted = true
	stopTimer = make(chan struct{})
	log.Print("Start timer")
	for countdown > 0 {
		select {
		default:
			countdown--
			<-tick
		case <-stopTimer:
			log.Print("Timer stopped")
			timerStarted = false
			return
		}
	}
	timerStarted = false
}

func displayHome(w http.ResponseWriter, r *http.Request) {
	if timerStarted {
		log.Print("Disable running timer from home page")
		close(stopTimer)
	}
	tmpl := template.Must(template.ParseFiles("templates/home.html"))
	data := Home{
		ButtonText: "Bienvenue dans le quizz",
		IntroURL:   introURL,
	}
	tmpl.Execute(w, data)
}

func displayIntro(w http.ResponseWriter, r *http.Request) {
	if timerStarted {
		log.Print("Disable running timer from intro page")
		close(stopTimer)
	}
	tmpl := template.Must(template.ParseFiles("templates/intro.html"))
	data := Intro{
		Video:            "intro.mp4",
		FirstQuestionURL: "/q/" + questions[0].ID,
	}
	tmpl.Execute(w, data)
}

func displayConclusion(w http.ResponseWriter, r *http.Request) {
	if timerStarted {
		log.Print("Disable running timer from success page")
		close(stopTimer)
	}
	tmpl := template.Must(template.ParseFiles("templates/conclusion.html"))
	data := Conclusion{
		Video: "endgame.mp4",
	}
	tmpl.Execute(w, data)
}

func displayEndDead(w http.ResponseWriter, r *http.Request) {
	log.Print("Disable running timer from failure page")
	close(stopTimer)
	tmpl := template.Must(template.ParseFiles("templates/dead.html"))
	tmpl.Execute(w, nil)
}

func getQuestion(w http.ResponseWriter, r *http.Request) {
	if !timerStarted {
		log.Print("The timer isn't started, go init timer")
		go initTimer()
	}

	params := mux.Vars(r)
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

func postAnswer(w http.ResponseWriter, r *http.Request) {
	proposal := r.FormValue("response")
	questionID := mux.Vars(r)["id"]

	question := getQuestionByID(questionID)
	if strings.ToUpper(question.Response) == strings.ToUpper(proposal) {
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

func readTimer(w http.ResponseWriter, r *http.Request) {
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
			if quotient == "0" {
				quotient = "  "
			} else if len(quotient) == 1 {
				quotient = " " + quotient
			}

			remainder := strconv.Itoa(countdown % 60)
			if len(remainder) == 1 {
				remainder = "0" + remainder
			}

			timeRemaining := quotient + ":" + remainder

			if err := conn.WriteMessage(1, []byte(timeRemaining)); err != nil {
				log.Println(err)
				return
			}
			time.Sleep(time.Second)
		}
		if err := conn.WriteMessage(1, []byte("OverTime")); err != nil {
			log.Println(err)
			return
		}
	}
}

func getQuestionByID(qID string) *Question {
	for _, q := range questions {
		if q.ID == qID {
			return q
		}
	}
	log.Printf("The question %s doesn't exists", qID)
	return nil
}

func getNextPage(currentID string) string {
	for i, q := range questions {
		if q.ID == currentID {
			if i == len(questions)-1 {
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
