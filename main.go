package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
)

type Student struct { //remember to name fields starting with capital letter too, otherwise they won't be visible to get marshalled
	Id    string `json:"ID"`
	Name  string `json:"Name"`
	Sem   uint8  `json:"Semester"`
	Marks uint8  `json:"Marks"`
}

var students map[string]Student
var creds map[string]string
var loggedin = make(map[string]string)

func InitializeDatabase() {
	creds = make(map[string]string)
	creds["user1"] = "passw0rd"
	creds["admin"] = "admin"

	students = make(map[string]Student)

	alpha := Student{
		Id:    "1001",
		Name:  "Alpha",
		Sem:   7,
		Marks: 97,
	}

	beta := Student{
		Id:    "1002",
		Name:  "Beta",
		Sem:   5,
		Marks: 98,
	}

	students[alpha.Id] = alpha
	students[beta.Id] = beta
}

func replyHome(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Student API")
}

func getStudents(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(students)
}

func getStudentById(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	student, ok := students[id]
	if ok {
		json.NewEncoder(w).Encode(student)
	} else {
		json.NewEncoder(w).Encode("Requested student not found")
	}
}

func setStudent(w http.ResponseWriter, r *http.Request) {
	sem, ok1 := strconv.ParseInt(r.FormValue("Semester"), 10, 8)
	marks, ok2 := strconv.ParseInt(r.FormValue("Marks"), 10, 8)
	if ok1 != nil || ok2 != nil {
		json.NewEncoder(w).Encode("Invalid fields")
		return
	}
	temp := Student{
		Id:    r.FormValue("ID"),
		Name:  r.FormValue("Name"),
		Sem:   uint8(sem),
		Marks: uint8(marks),
	}
	students[temp.Id] = temp
}

func deleteStudent(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if _, ok := students[id]; ok {
		delete(students, id)
		w.Write([]byte("Student with ID: " + id + " deleted"))
	} else {
		w.Write([]byte("Student with ID: " + id + " does not exist"))
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	sid, err := r.Cookie("SessionID")
	if err == nil {
		if _, ok := loggedin[sid.Value]; ok {
			fmt.Println("Already logged in")
			http.Redirect(w, r, "/students", 302)
			return
		}
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	fmt.Println("POSTED: ", username, password)
	pwd, ok := creds[username]
	if ok && password == pwd {
		token := strconv.FormatInt(rand.Int63(), 10)
		loggedin[token] = username
		http.SetCookie(w, &http.Cookie{
			Name:   "SessionID",
			Value:  token,
			MaxAge: 100,
		})
		http.Redirect(w, r, "/students", 302)
	} else {
		http.Redirect(w, r, "/login", 302)
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	sid, err := r.Cookie("SessionID")
	if err == nil {
		delete(loggedin, sid.Value)
	}
	fmt.Fprintln(w, "Logged out")
}

type Decorator func(handler http.Handler) http.Handler

func logger() Decorator {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			fmt.Println(request.URL)
			handler.ServeHTTP(writer, request)
		})
	}
}

func trailer() Decorator {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			handler.ServeHTTP(writer, request)
			fmt.Fprintln(writer, "--- END ---")
		})
	}
}

func authenticate() Decorator {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			sid, err := request.Cookie("SessionID")
			if err == nil {
				fmt.Println(loggedin)
				usr, ok := loggedin[sid.Value]
				fmt.Println(usr, ok)
				if ok {
					handler.ServeHTTP(writer, request)
				} else {
					http.Redirect(writer, request, "/login", 302)
				}
			} else {
				http.Redirect(writer, request, "/login", 302)
			}
		})
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	sid, err := r.Cookie("SessionID")
	if err == nil {
		if _, ok := loggedin[sid.Value]; ok {
			fmt.Println("Already logged in")
			http.Redirect(w, r, "/students", 302)
			return
		}
	}
	buff, err := ioutil.ReadFile("./login.html")
	if err == nil {
		w.Write(buff)
	}
}

func Adapt(handler http.Handler, decorators ...Decorator) http.Handler {
	for _, decorator := range decorators {
		handler = decorator(handler)
	}
	return handler
}

func main() {
	InitializeDatabase()
	handleRequests()
}

func handleRequests() {
	router := mux.NewRouter()
	router.HandleFunc("/home", home).Methods("POST")
	router.HandleFunc("/", replyHome).Methods("GET")
	router.Handle("/students", Adapt(http.HandlerFunc(getStudents), trailer(), authenticate())).Methods("GET")
	router.HandleFunc("/students/{id}", getStudentById).Methods("GET")
	router.HandleFunc("/students/add", setStudent).Methods("POST")
	router.HandleFunc("/students/delete/{id}", deleteStudent).Methods("DELETE")
	router.HandleFunc("/login", login)
	router.HandleFunc("/logout", logout)
	handler := Adapt(router, logger())

	fmt.Println("Server started")
	log.Fatal(http.ListenAndServe(":8090", handler))
}
