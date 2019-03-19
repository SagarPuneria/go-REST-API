package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	SqlInterface "go-REST-API/sqlinterface"
	ut "go-REST-API/util"

	_ "github.com/go-sql-driver/mysql"
	mx "github.com/gorilla/mux"
)

type app struct {
	router                          *mx.Router
	sqlDB                           *SqlInterface.MySqldb
	bl                              bool
	previousLayout, previousSeating string
}

type customer struct {
	PartySize int    `json:"party_size"`
	Phone     int    `json:"phone"`
	Name      string `json:"name"`
}
type booking struct {
	TableID     int    `json:"table_id"`
	BookingDate string `json:"booking_date"`
	BookingTime string `json:"booking_time"`
	NoOfSeats   int    `json:"no_of_seats"`
	cus         customer
}
type layoutSeating struct {
	TableID       int
	MaxSeating    int
	MinOccupency  int
	BookingStatus bool
}
type layout struct {
	TableID    int `json:"table_id"`
	MaxSeating int `json:"max_seating"`
}
type seating struct {
	MinOccupency    int `json:"min_occupency"`
	SeatingCapacity int `json:"seating_capacity"`
}
type avgTOT struct {
	TableID int
	avgTime <-chan time.Time
}

var avgTots []avgTOT
var layoutSeatings []layoutSeating
var rWMutex sync.RWMutex

func main() {
	defer func() {
		if errD := recover(); errD != nil {
			fmt.Println("Exception occurred at ", ut.RecoverExceptionDetails(ut.FunctionName()), " and recovered in main function, Error Info: ", errD)
		}
	}()
	user := os.Args[1]
	pass := os.Args[2]
	host := os.Args[3]
	port := os.Args[4]
	a := app{}
	a.sqlDB = a.initializeDB(user, pass, host, port)
	defer a.sqlDB.Close()
	a.router = mx.NewRouter()
	a.initializeRoutes()
	go a.readFromFile("layout.json", "seating.json")

	a.run("localhost:8080")
}

func (a *app) initializeDB(user, pass, host, port string) *SqlInterface.MySqldb {
	defer func() {
		if errD := recover(); errD != nil {
			fmt.Println("Exception occurred at ", ut.RecoverExceptionDetails(ut.FunctionName()), " and recovered in initializeDB method, Error Info: ", errD)
		}
	}()
	DNS := user + ":" + pass + "@tcp(" + host + ":" + port + ")/"
	createDB := "CREATE DATABASE IF NOT EXISTS McDoeRestaurant;"
	useDB := "USE McDoeRestaurant;"
	table := "CREATE TABLE IF NOT EXISTS booking (TableID int NOT NULL, Booking_date varchar(255), Booking_time varchar(255), No_of_seats int, PRIMARY KEY (TableID));"
	sqlDB, err := SqlInterface.CreateDataBase(DNS, createDB, useDB, table)
	if err != nil {
		log.Fatal(err)
	}
	return sqlDB
}

func (a *app) readFromFile(filename1, filename2 string) {
	defer func() {
		if errD := recover(); errD != nil {
			fmt.Println("Exception occurred at ", ut.RecoverExceptionDetails(ut.FunctionName()), " and recovered in readFromFile method, Error Info: ", errD)
		}
	}()
	for {
		var currentLayout, currentSeating string
		jsondata, err := ioutil.ReadFile(filename1)
		if err != nil {
			log.Fatal("ioutil.ReadFile:", err.Error())
		}
		currentLayout = string(jsondata)
		var layouts []layout
		err = json.Unmarshal(jsondata, &layouts)
		if err != nil {
			log.Fatal("error:", err)
		}
		jsondata, err = ioutil.ReadFile(filename2)
		if err != nil {
			log.Fatal("ioutil.ReadFile:", err.Error())
		}
		currentSeating = string(jsondata)
		var seatings []seating
		err = json.Unmarshal(jsondata, &seatings)
		if err != nil {
			log.Fatal("error:", err)
		}
		var prevLayoutSeatings []layoutSeating
		if a.previousLayout != "" && a.previousSeating != "" && (a.previousLayout != currentLayout || a.previousSeating != currentSeating) {
			prevLayoutSeatings = layoutSeatings
			rWMutex.Lock()
			layoutSeatings = nil
			rWMutex.Unlock()
		}
		if layoutSeatings == nil {
			a.updateLayoutSeatings(layouts, seatings, prevLayoutSeatings, &layoutSeatings)
		}
		a.previousLayout = currentLayout
		a.previousSeating = currentSeating
		time.Sleep(1 * time.Minute)
	}
}

func (a *app) updateLayoutSeatings(layouts []layout, seatings []seating, prevLayoutSeatings []layoutSeating, layoutSeatings *[]layoutSeating) {
	defer func() {
		if errD := recover(); errD != nil {
			fmt.Println("Exception occurred at ", ut.RecoverExceptionDetails(ut.FunctionName()), " and recovered in updateLayoutSeatings method, Error Info: ", errD)
		}
	}()
	rWMutex.Lock()
	defer rWMutex.Unlock()
	for _, layout := range layouts {
		var ls layoutSeating
		ls.TableID = layout.TableID
		ls.MaxSeating = layout.MaxSeating
		ls.MinOccupency = ls.MaxSeating
		for _, seating := range seatings {
			if layout.MaxSeating == seating.SeatingCapacity {
				ls.MinOccupency = seating.MinOccupency
			}
		}
		strQuery := "SELECT TableID FROM booking WHERE TableID=?"
		ID, _ := a.sqlDB.SelectQueryRow(strQuery, ls.TableID)
		if ID == ls.TableID {
			ls.BookingStatus = true
		}
		*layoutSeatings = append(*layoutSeatings, ls)
	}
	if prevLayoutSeatings != nil {
		for i, layoutSeating := range *layoutSeatings {
			for _, prevLayoutSeating := range prevLayoutSeatings {
				if layoutSeating.TableID == prevLayoutSeating.TableID {
					(*layoutSeatings)[i].BookingStatus = prevLayoutSeating.BookingStatus
					if (prevLayoutSeating.BookingStatus == true) && ((layoutSeating.MaxSeating != prevLayoutSeating.MaxSeating) || (layoutSeating.MinOccupency != prevLayoutSeating.MinOccupency)) {
						// Remove record from DB
						strQuery := fmt.Sprintf("DELETE FROM booking WHERE TableID='%d';", prevLayoutSeating.TableID)
						err := a.sqlDB.ExecuteQuery(strQuery)
						if err != nil {
							fmt.Println("DELETE Execution error:", err)
						} else {
							(*layoutSeatings)[i].BookingStatus = false
						}
					}
				}
			}
		}
	}
}

func (a *app) run(addr string) {
	defer func() {
		if errD := recover(); errD != nil {
			fmt.Println("Exception occurred at ", ut.RecoverExceptionDetails(ut.FunctionName()), " and recovered in run method, Error Info: ", errD)
		}
	}()
	log.Fatal(http.ListenAndServe(addr, a.router))
}

func (a *app) initializeRoutes() {
	defer func() {
		if errD := recover(); errD != nil {
			fmt.Println("Exception occurred at ", ut.RecoverExceptionDetails(ut.FunctionName()), " and recovered in initializeRoutes method, Error Info: ", errD)
		}
	}()
	go func() {
		for {
			greetings := "Enter your choice:\n\t1. Open restaurant.\n\t2. Close restuarant.\n\t3. Exit"
			fmt.Println(greetings)
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			choiceInteger, err := strconv.Atoi(scanner.Text())
			if err != nil {
				fmt.Println("Error in choice input and Error Info:", err)
				fmt.Println("Enter correct integer number")
				continue
			}
			switch choiceInteger {
			case 1:
				a.bl = true
			case 2:
				a.bl = false
			case 3:
				os.Exit(0)
			default:
				fmt.Println("Invalid choice")
			}
		}
	}()
	//date:year-month-day
	//time:HH:MM:SS
	a.router.HandleFunc("/booking", a.getBooking).Methods("GET")
	a.router.HandleFunc("/booking/{date:[0-9]+[0-9]+[0-9]+[0-9]+-[0-9]+[0-9]+-[0-9]+[0-9]+}/{time:[0-9]+[0-9]+:[0-9]+[0-9]+:[0-9]+[0-9]+}", a.createBooking).Methods("POST")
	go a.checkAverageTableOccupancyTimes()
}

func (a *app) getBooking(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if errD := recover(); errD != nil {
			fmt.Println("Exception occurred at ", ut.RecoverExceptionDetails(ut.FunctionName()), " and recovered in getBooking method, Error Info: ", errD)
		}
	}()
	if !a.bl {
		respondWithError(w, http.StatusInternalServerError, "Restaurant closed, try after some time")
		return
	}
	respondWithJSON(w, http.StatusOK, layoutSeatings)
}

func (a *app) checkAverageTableOccupancyTimes() {
	for {
		time.Sleep(500 * time.Millisecond)
		for i, avgTot := range avgTots {
			select {
			case <-avgTot.avgTime:
				for j, layoutSeating := range layoutSeatings {
					if layoutSeating.TableID == avgTot.TableID {
						// Remove record from DB
						strQuery := fmt.Sprintf("DELETE FROM booking WHERE TableID='%d';", layoutSeating.TableID)
						err := a.sqlDB.ExecuteQuery(strQuery)
						if err != nil {
							fmt.Println("DELETE Execution error:", err)
						} else {
							layoutSeatings[j].BookingStatus = false
							avgTots = append(avgTots[:i], avgTots[i+1:]...)
						}
						break
					}
				}
			}
		}
	}
}

func (a *app) createBooking(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if errD := recover(); errD != nil {
			fmt.Println("Exception occurred at ", ut.RecoverExceptionDetails(ut.FunctionName()), " and recovered in createBooking method, Error Info: ", errD)
		}
	}()
	if !a.bl {
		respondWithError(w, http.StatusInternalServerError, "Restaurant closed, try after some time")
		return
	}
	params := mx.Vars(r)
	strDate := params["date"]
	strTime := params["time"]
	t, err := time.Parse(time.RFC3339, strDate+"T"+strTime+"Z")
	if err != nil {
		fmt.Println("time.Parse error:", err)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	now := time.Now().UTC()
	difference := t.Sub(now)
	//if difference.Hours() < 2 || difference.Hours() > 48 {
	if difference.Hours() > 48 {
		respondWithError(w, http.StatusInternalServerError, "Booking request should be made 2 to 48 hrs prior")
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Println("ioutil.ReadAll error:", err)
	}
	var c customer
	if err = json.NewDecoder(strings.NewReader(string(body))).Decode(&c); err != nil {
		fmt.Println("json.NewDecoder error:", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()
	var b booking
	rWMutex.Lock()
	for i, layoutSeating := range layoutSeatings {
		if !layoutSeating.BookingStatus && layoutSeating.MaxSeating != 0 && layoutSeating.MinOccupency != 0 {
			if c.PartySize <= layoutSeating.MaxSeating && c.PartySize >= layoutSeating.MinOccupency {
				b.TableID = layoutSeating.TableID
				b.NoOfSeats = layoutSeating.MaxSeating
				layoutSeatings[i].BookingStatus = true
				break
			}
		}
	}
	rWMutex.Unlock()
	if b.TableID == 0 || b.NoOfSeats == 0 {
		respondWithError(w, http.StatusInternalServerError, "Booking is full, try after some time")
		return
	}
	b.cus = c
	b.BookingDate = params["date"]
	b.BookingTime = params["time"]
	if err := b.createBooking(a.sqlDB); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	getCheckAverageTableOccupancyTimes(c, b.TableID)
	respondWithJSON(w, http.StatusCreated, map[string]interface{}{"success": true, "result": b})
}

func getCheckAverageTableOccupancyTimes(c customer, tableID int) {
	jsondata, err := ioutil.ReadFile("tot.json")
	if err != nil {
		log.Fatal("ioutil.ReadFile:", err.Error())
	}
	type tot struct {
		MinPartySize int    `json:"min_party_size"`
		MaxPartySize int    `json:"max_party_size"`
		AvgTot       string `json:"avg_tot"`
	}
	var tots []tot
	err = json.Unmarshal(jsondata, &tots)
	if err != nil {
		log.Fatal("error:", err)
	}
	var avgtot avgTOT
	for _, t := range tots {
		if c.PartySize >= t.MinPartySize && t.MaxPartySize == 0 {
			freq, _ := strconv.Atoi(strings.TrimRight(t.AvgTot, "m"))
			avgtot.avgTime = time.Tick(time.Duration(freq) * time.Minute)
			avgtot.TableID = tableID
			avgTots = append(avgTots, avgtot)
		} else if c.PartySize >= t.MinPartySize && c.PartySize <= t.MaxPartySize {
			freq, _ := strconv.Atoi(strings.TrimRight(t.AvgTot, "m"))
			avgtot.avgTime = time.Tick(time.Duration(freq) * time.Minute)
			avgtot.TableID = tableID
			avgTots = append(avgTots, avgtot)
		}
	}
}

func (b *booking) createBooking(db *SqlInterface.MySqldb) error {
	defer func() {
		if errD := recover(); errD != nil {
			fmt.Println("Exception occurred at ", ut.RecoverExceptionDetails(ut.FunctionName()), " and recovered in createBooking method, Error Info: ", errD)
		}
	}()
	strQuery := fmt.Sprintf("INSERT INTO booking (TableID, Booking_date, Booking_time, No_of_seats) VALUES('%d', '%s', '%s', '%d');", b.TableID, b.BookingDate, b.BookingTime, b.NoOfSeats)
	err := db.ExecuteQuery(strQuery)
	if err != nil {
		fmt.Println("INSERT INTO Execution error:", err)
		rWMutex.Lock()
		for i, layoutSeating := range layoutSeatings {
			if layoutSeating.TableID == b.TableID {
				layoutSeatings[i].BookingStatus = false
				break
			}
		}
		rWMutex.Unlock()
		return err
	}
	return nil
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	defer func() {
		if errD := recover(); errD != nil {
			fmt.Println("Exception occurred at ", ut.RecoverExceptionDetails(ut.FunctionName()), " and recovered in respondWithError function, Error Info: ", errD)
		}
	}()
	respondWithJSON(w, code, map[string]interface{}{"success": false, "errors": map[string]string{"reason": message}})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	defer func() {
		if errD := recover(); errD != nil {
			fmt.Println("Exception occurred at ", ut.RecoverExceptionDetails(ut.FunctionName()), " and recovered in respondWithJSON function, Error Info: ", errD)
		}
	}()
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
