package render

import (
	"math"
	"sort"
)

const campMaxFigures = 20

type campPerson struct {
	position [4]float64 // x, seated y, perspective scale, facing direction
	pose     int
}

func campFigureCount(population uint32) int {
	return int(min(uint32(campMaxFigures), population))
}

func campCrowd(population uint32) []campPerson {
	count := campFigureCount(population)
	people := make([]campPerson, 0, count)
	// Fill opposite sides in pairs; the original eight seats keep their poses.
	seats := [8][4]float64{{446, 423, .86, 1}, {545, 394, .76, 1}, {737, 397, .79, -1}, {843, 430, .90, -1}, {343, 494, 1.15, 1}, {944, 491, 1.12, -1}, {486, 551, 1.27, 1}, {795, 551, 1.30, -1}}
	for _, index := range [8]int{0, 3, 1, 2, 4, 5, 6, 7} {
		if len(people) == count {
			break
		}
		people = append(people, campPerson{position: seats[index], pose: index})
	}
	// Additional seats sit farther from the viewer and behind the main ring.
	// Every pose remains inside the gathering's existing animation crop.
	columns := count - len(people)
	for column := range columns {
		fraction := float64(column+1) / float64(columns+1)
		x := 310 + 660*fraction
		y := 332 - 12*math.Sin(fraction*math.Pi)
		direction := 1.0
		if x > 640 {
			direction = -1
		}
		people = append(people, campPerson{position: [4]float64{x, y, .46, direction}, pose: 8 + column})
	}
	sort.Slice(people, func(i, j int) bool { return people[i].position[1] < people[j].position[1] })
	return people
}
