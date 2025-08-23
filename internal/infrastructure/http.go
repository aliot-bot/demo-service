package infrastructure

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type Server struct {
	cache  *Cache
	store  *Postgres
	router *mux.Router
}

func NewServer(cache *Cache, store *Postgres) *Server {
	s := &Server{
		cache:  cache,
		store:  store,
		router: mux.NewRouter(),
	}
	s.router.HandleFunc("/order/{order_uid}", s.handleGetOrder).Methods("GET")
	s.router.HandleFunc("/", s.handleUserOrder).Methods("GET")
	s.router.HandleFunc("/order_search", s.handleUserOrder).Methods("GET")
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
