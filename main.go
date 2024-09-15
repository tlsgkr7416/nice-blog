package main

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Board struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Title     string
	Author    string
	Content   string
}

type HealthCheckResponse struct {
	Status string `json:"status"`
}

var (
	tpl    *template.Template
	gormDB *gorm.DB
	//go:embed web
	staticContent embed.FS

	// Prometheus metrics
	writeCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "write_requests_total",
			Help: "Total number of write requests",
		},
		[]string{"status"},
	)
	boardCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "board_requests_total",
			Help: "Total number of board requests",
		},
		[]string{"status"},
	)
)

const (
	MaxPerPage = 5
)

func init() {
	// tpl = template.Must(template.ParseGlob("web/templates/*.gohtml"))
	tpl = template.Must(template.ParseFS(staticContent, "web/templates/*"))

	// Register metrics
	prometheus.MustRegister(writeCount)
	prometheus.MustRegister(boardCount)
}

func main() {
	srv := &http.Server{Addr: ":8080"}

	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	var connectionString = fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4&parseTime=True", "root", "1234", "mysql-0.mysql.default", "test")
	mysqlDB, err := sql.Open("mysql", connectionString)
	defer mysqlDB.Close()

	gormDB, err = gorm.Open(mysql.New(mysql.Config{
		Conn: mysqlDB,
	}), &gorm.Config{})

	if err != nil {
		panic("failed to connect database")
	}

	gormDB.AutoMigrate(&Board{})

	http.HandleFunc("/", index)
	http.HandleFunc("/write", write)
	http.HandleFunc("/board/", board)
	http.HandleFunc("/delete/", delete)
	http.HandleFunc("/edit/", edit)
	// Prometheus metrics endpoint
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", healthCheck)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP 서버 에러: %s\n", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM)
	<-quit
	fmt.Println("서버가 종료되고 있습니다...")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("서버 셧다운 에러: %s\n", err)
	}

	fmt.Println("서버가 안전하게 종료되었습니다.")
}

func index(w http.ResponseWriter, r *http.Request) {
	tpl.ExecuteTemplate(w, "index.gohtml", nil)
}
func healthCheck(w http.ResponseWriter, r *http.Request) {
	response := HealthCheckResponse{Status: "ok"}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func write(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodGet {
		tpl.ExecuteTemplate(w, "write.gohtml", nil)
	}

	if r.Method == http.MethodPost {
		title := r.PostFormValue("title")
		author := r.PostFormValue("author")
		content := r.PostFormValue("content")

		newPost := Board{Title: title, Author: author, Content: content}
		gormDB.Create(&newPost)
		writeCount.WithLabelValues("success").Inc()

		http.Redirect(w, r, "/", http.StatusSeeOther)

	}

	log.Println("write success")

}

func board(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var boards []Board
	if result := gormDB.Find(&boards); result.Error != nil {
		http.Error(w, "Error fetching data", http.StatusInternalServerError)
		return
	}

	err := tpl.ExecuteTemplate(w, "boardList.gohtml", boards)
	if err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}
	boardCount.WithLabelValues("success").Inc()
	log.Println("Successfully read the board list")
}

func delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/delete/")
	gormDB.Delete(&Board{}, id)

	http.Redirect(w, r, "/board", http.StatusSeeOther)
}

func edit(w http.ResponseWriter, r *http.Request) {

	id := strings.TrimPrefix(r.URL.Path, "/edit/")
	var b Board

	gormDB.First(&b, id)

	if r.Method == http.MethodPost {

		gormDB.Model(&b).Updates(Board{Title: r.PostFormValue("title"), Author: r.PostFormValue("author"), Content: r.PostFormValue("content")})
		http.Redirect(w, r, "/board", http.StatusSeeOther)
		return
	}

	tpl.ExecuteTemplate(w, "write.gohtml", b)
}
