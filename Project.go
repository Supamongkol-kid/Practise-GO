package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type booking struct {
	BookingId          int    `json: "bookingid"`
	BookingTime        string `json: "bookingtime" gorm:"type:timestamp"`
	BookingClassroomId string `json: "bookingclassroomid"`
	BookingBookerId    string `json: "bookingbookerid"`
}

var Db *sql.DB

const bookerPath = "booker"
const bookingPath = "bookings"
const basePath = "/api"

func getBooking(bookingId int) (*booking, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	row := Db.QueryRowContext(ctx, `SELECT booking_id, booking_time, booking_classroom_id, booking_student_id FROM booking WHERE booking_id = ?`, bookingId)
	booking := &booking{}
	err := row.Scan(&booking.BookingId, &booking.BookingTime, &booking.BookingClassroomId, &booking.BookingBookerId)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		log.Println(err)
		return nil, err
	}
	return booking, nil
}

func getBooker(bookerId string) ([]booking, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	results, err := Db.QueryContext(ctx, `SELECT booking_id, booking_time, booking_classroom_id, booking_student_id FROM booking WHERE booking_student_id = ?`, bookerId)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer results.Close()
	booker := make([]booking, 0)
	for results.Next() {
		var bookers booking
		results.Scan(&bookers.BookingId, &bookers.BookingTime, &bookers.BookingClassroomId, &bookers.BookingBookerId)
		booker = append(booker, bookers)
	}
	return booker, nil
}

func getBookingList() ([]booking, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	results, err := Db.QueryContext(ctx, `SELECT booking_id, booking_time, booking_classroom_id, booking_student_id FROM booking`)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer results.Close()
	bookings := make([]booking, 0)
	for results.Next() {
		var booking booking
		results.Scan(&booking.BookingId, &booking.BookingTime, &booking.BookingClassroomId, &booking.BookingBookerId)
		bookings = append(bookings, booking)
	}
	return bookings, nil
}

func insertBooking(booking booking) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := Db.ExecContext(ctx, `INSERT INTO booking (booking_time, booking_classroom_id, booking_student_id) VALUES (?, ?, ?)`, booking.BookingTime, booking.BookingClassroomId, booking.BookingBookerId)
	if err != nil {
		log.Println(err.Error())
		return 0, err
	}
	insertId, err := result.LastInsertId()
	if err != nil {
		log.Println(err.Error())
		return 0, err
	}
	return int(insertId), nil
}

func removeBooking(bookingId int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := Db.ExecContext(ctx, `DELETE FROM booking WHERE booking_id = ?`, bookingId)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}

func handlerBooking(w http.ResponseWriter, r *http.Request) {
	urlPathSegments := strings.Split(r.URL.Path, fmt.Sprintf("%s/", bookingPath))
	if len(urlPathSegments[1:]) > 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	bookingId, err := strconv.Atoi(urlPathSegments[len(urlPathSegments)-1])
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	switch r.Method {
	case http.MethodGet:
		booking, err := getBooking(bookingId)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if booking == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		j, err := json.Marshal(booking)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, err = w.Write(j)
		if err != nil {
			log.Fatal(err)
		}
	case http.MethodDelete:
		err := removeBooking(bookingId)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handlerBooker(w http.ResponseWriter, r *http.Request) {
	urlPathSegments := strings.Split(r.URL.Path, fmt.Sprintf("%s/", bookerPath))
	urlPathSegments = strings.Split(strings.Join(urlPathSegments, ""), "/")
	bookerId := urlPathSegments[2]
	switch r.Method {
	case http.MethodGet:
		booker, err := getBooker(bookerId)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		j, err := json.Marshal(booker)
		if err != nil {
			log.Fatal(err)
		}
		_, err = w.Write(j)
		if err != nil {
			log.Fatal(err)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handlerBookings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		bookingList, err := getBookingList()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		j, err := json.Marshal(bookingList)
		if err != nil {
			log.Fatal(err)
		}
		_, err = w.Write(j)
		if err != nil {
			log.Fatal(err)
		}
	case http.MethodPost:
		var booking booking
		err := json.NewDecoder(r.Body).Decode(&booking)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		bookingId, err := insertBooking(booking)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(fmt.Sprintf(`{"bookingid":%d}`, bookingId)))
	case http.MethodOptions:
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func corsMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Add("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Authorization, X-Custom-Header")
		handler.ServeHTTP(w, r)
	})
}

func setupRoutes(apiBasePath string) {
	bookingHandler := http.HandlerFunc(handlerBooking)
	http.Handle(fmt.Sprintf("%s/%s/", apiBasePath, bookingPath), corsMiddleware(bookingHandler))
	bookingsHandler := http.HandlerFunc(handlerBookings)
	http.Handle(fmt.Sprintf("%s/%s", apiBasePath, bookingPath), corsMiddleware(bookingsHandler))
	bookerHandler := http.HandlerFunc(handlerBooker)
	http.Handle(fmt.Sprintf("%s/%s/", apiBasePath, bookerPath), corsMiddleware(bookerHandler))
}

func setupDb() {
	var err error
	Db, err = sql.Open("mysql", "root:141453@tcp(127.0.0.1:3306)/classroom")
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("Connect successfully!!!")
	}
	//fmt.Println(Db)
	Db.SetConnMaxLifetime(time.Minute * 3)
	Db.SetMaxOpenConns(10)
	Db.SetMaxIdleConns(10)
}

func main() {
	setupDb()
	setupRoutes(basePath)
	log.Fatal(http.ListenAndServe(":5000", nil))
}
