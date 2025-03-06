package models

import (
	"encoding/json"

	"time"

	_ "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type User struct {
	ID          uint      `gorm:"primaryKey;" json:"id"`
	UserName    string    `gorm:"size:255;not null" json:"username"`
	Email       string    `gorm:"size:255;unique;not null" json:"email"`
	Password    string    `gorm:"size:255;not null" json:"-"`
	WatchList   []Film    `gorm:"many2many:user_watchlist" json:"watchlist"`
	WatchedList []Film    `gorm:"many2many:user_watchedlist" json:"watchedlist"`
	Created     time.Time `gorm:"autoCreateTime" json:"created"`
}

type Genre struct {
	ID   uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Name string `gorm:"size:255;not null" json:"name"`
}

type Star struct {
	ID   uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Name string `gorm:"size:255;not null" json:"name"`
}

type Director struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Name string `gorm:"size:255;not null" json:"name"`
}

type Film struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"size:255;not null" json:"name"`
	Year        int        `gorm:"not null" json:"year"`
	RunTime     int        `gorm:"not null" json:"runtime"`
	Rating      float32    `gorm:"not null" json:"rating"`
	Genres      []Genre    `gorm:"many2many:film_genres" json:"genres"`
	Directors   []Director `gorm:"many2many:film_directors" json:"directors"`
	Stars       []Star     `gorm:"many2many:film_stars" json:"stars"`
	Description string     `gorm:"type:text" json:"description"`
	Image       string     `gorm:"size:255" json:"image"`
}

type FilmWithUsers struct {
	Film
	Users    []string `json:"users"`
	Watchers []string `json:"watchers"`
}

func (f *Film) Json(db *gorm.DB) ([]byte, error) {
	var users []User
	var watchers []User

	db.Model(&f).Association("WatchList").Find(&users)
	db.Model(&f).Association("WatchedList").Find(&watchers)

	userNames := make([]string, len(users))
	for i, user := range users {
		userNames[i] = user.UserName
	}

	watcherNames := make([]string, len(watchers))
	for i, watcher := range watchers {
		watcherNames[i] = watcher.UserName
	}

	filmWithUsers := FilmWithUsers{
		Film:     *f,
		Users:    userNames,
		Watchers: watcherNames,
	}

	return json.Marshal(filmWithUsers)
}
