package dataloader

import (
	"encoding/json"
	"errors"
	"os"

	"gorm.io/gorm"
	"movies4u.net/internals/models"
)

type DataLoader struct {
	DB *gorm.DB
}

var ErrDataLoaded = errors.New("DATABASE LOADED")

type FilmData struct {
	Name        string   `json:"name"`
	Year        int      `json:"year"`
	RunTime     int      `json:"runtime"`
	Rating      float32  `json:"rating"`
	Genres      []string `json:"genre"`
	Image       string   `json:"image"`
	Description string   `json:"description"`
	Director    string   `json:"director"`
	Stars       []string `json:"stars"`
	ID          uint     `json:"id"`
}

func (dl *DataLoader) LoadFilmsFromFile(filePath string) error {
	var filmCount int64;
	dl.DB.Model(&models.Film{}).Count(&filmCount)
	if filmCount == 9999{
		return ErrDataLoaded
	}
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var filmsData []FilmData
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&filmsData)
	if err != nil {
		return err
	}

	if len(filmsData) == 0 {
		return errors.New("empty slice found")
	}

	genresSet := make(map[string]struct{})
	directorsSet := make(map[string]struct{})
	starsSet := make(map[string]struct{})
	
	for _, filmData := range filmsData {

		for _, genre := range filmData.Genres {
			
			genresSet[genre] = struct{}{}
		}
		directorsSet[filmData.Director] = struct{}{}
		for _, star := range filmData.Stars {
			starsSet[star] = struct{}{}
		}
	}

	if len(genresSet) == 0 {
		return errors.New("no genres found")
	}
	if len(directorsSet) == 0 {
		return errors.New("no directors found")
	}
	if len(starsSet) == 0 {
		return errors.New("no stars found")
	}

	var genres []models.Genre
	for genre := range genresSet {
		genres = append(genres, models.Genre{Name: genre})
	}
	err = dl.DB.Create(&genres).Error
	if err != nil {
		return err
	}

	var directors []models.Director
	for director := range directorsSet {
		directors = append(directors, models.Director{Name: director})
	}
	err = dl.DB.Create(&directors).Error
	if err != nil {
		return err
	}

	var stars []models.Star
	for star := range starsSet {
		stars = append(stars, models.Star{Name: star})
	}
	err = dl.DB.Create(&stars).Error
	if err != nil {
		return err
	}

	for _, filmData := range filmsData {
	
		var filmGenres []models.Genre
		for _, genreName := range filmData.Genres {
			var genre models.Genre
			err = dl.DB.Where("name = ?", genreName).First(&genre).Error
			if err != nil {
				return err
			}
			filmGenres = append(filmGenres, genre)
		}

		var director models.Director
		err = dl.DB.Where("name = ?", filmData.Director).First(&director).Error
		if err != nil {
			return err
		}

		var filmStars []models.Star
		for _, starName := range filmData.Stars {
			var star models.Star
			err = dl.DB.Where("name = ?", starName).First(&star).Error
			if err != nil {
				return err
			}
			filmStars = append(filmStars, star)
		}

		film := models.Film{
			ID:          filmData.ID,
			Name:        filmData.Name,
			Year:        filmData.Year,
			RunTime:     filmData.RunTime,
			Rating:      filmData.Rating,
			Genres:      filmGenres,
			Directors:   []models.Director{director},
			Stars:       filmStars,
			Description: filmData.Description,
			Image:       filmData.Image,
		}

		err = dl.DB.Create(&film).Error
		if err != nil {
			return err
		}
	}

	return nil
}
