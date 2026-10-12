package main

import (
	"database/sql"
	"net/http"
)

type GetUserHandler struct {
	db *sql.DB
}

func NewGetUserHandler(db *sql.DB) *GetUserHandler {
	return &GetUserHandler{
		db: db,
	}
}

func (h GetUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

}
