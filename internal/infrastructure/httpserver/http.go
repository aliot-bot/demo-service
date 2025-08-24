package httpserver

import (
	"encoding/json"
	"log"
	"net/http"

	"demo-service/internal/infrastructure/cache"
	"demo-service/internal/infrastructure/postgres"

	"github.com/gorilla/mux"
)

type Server struct {
	cache  *cache.Cache
	store  *postgres.Postgres
	router *mux.Router
}

func NewServer(cacheStore *cache.Cache, store *postgres.Postgres) *Server {
	s := &Server{
		cache:  cacheStore,
		store:  store,
		router: mux.NewRouter(),
	}
	s.router.HandleFunc("/order/{order_uid}", s.handleGetOrder).Methods("GET")
	s.router.HandleFunc("/", s.handleUserOrder).Methods("GET")
	return s
}

func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	uid := mux.Vars(r)["order_uid"]

	if order, ok := s.cache.Get(uid); ok {
		writeJSON(w, order)
		return
	}

	order, err := s.store.GetOrder(r.Context(), uid)
	if err != nil {
		log.Println("Заказ не найден:", uid, err)
		http.Error(w, "Заказ не найден", http.StatusNotFound)
		return
	}

	s.cache.Set(order)
	writeJSON(w, order)
}

func (s *Server) handleUserOrder(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/index.html")
}

func (s *Server) Start(addr string) error {
	log.Println("Сервер запущен на", addr)
	return http.ListenAndServe(addr, s.router)
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Println("Ошибка при записи JSON:", err)
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
	}
}
