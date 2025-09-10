package main

import (
	"encoding/xml"
	"fmt"
	"math"
	"os"
)

// GPX represents the root of a GPX file
type GPX struct {
	XMLName xml.Name `xml:"gpx"`
	Tracks  []Track  `xml:"trk"`
}

// Track represents a <trk> element in GPX
type Track struct {
	XMLName   xml.Name       `xml:"trk"`
	Name      string         `xml:"name"`
	TrackSegs []TrackSegment `xml:"trkseg"`
}

// TrackSegment represents a <trkseg> element in GPX
type TrackSegment struct {
	XMLName     xml.Name     `xml:"trkseg"`
	TrackPoints []TrackPoint `xml:"trkpt"`
}

// TrackPoint represents a <trkpt> element in GPX
type TrackPoint struct {
	XMLName   xml.Name `xml:"trkpt"`
	Latitude  float64  `xml:"lat,attr"`
	Longitude float64  `xml:"lon,attr"`
	Elevation float64  `xml:"ele"` // Elevation in meters
}

// earthRadius is the average radius of the Earth in kilometers
const earthRadius = 6371.0

// toRadians converts degrees to radians
func toRadians(deg float64) float64 {
	return deg * math.Pi / 180.0
}

func haversineDistance2D(p1, p2 TrackPoint) float64 {
	lat1 := toRadians(p1.Latitude)
	lon1 := toRadians(p1.Longitude)
	lat2 := toRadians(p2.Latitude)
	lon2 := toRadians(p2.Longitude)

	dLat := lat2 - lat1
	dLon := lon2 - lon1

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c // in km
}

func applyMovingAverage(points []TrackPoint, windowSize int) []TrackPoint {
	if windowSize < 1 || len(points) == 0 {
		return points
	}
	if windowSize%2 == 0 {
		windowSize++
	}

	smoothedPoints := make([]TrackPoint, len(points))
	halfWindow := windowSize / 2

	for i := range points {
		smoothedPoints[i] = points[i]

		sumElevation := 0.0
		count := 0

		start := i - halfWindow
		if start < 0 {
			start = 0
		}
		end := i + halfWindow
		if end >= len(points) {
			end = len(points) - 1
		}

		for j := start; j <= end; j++ {
			sumElevation += points[j].Elevation
			count++
		}
		smoothedPoints[i].Elevation = sumElevation / float64(count)
	}
	return smoothedPoints
}

func calculateCumulativeAscent(points []TrackPoint, threshold float64) float64 {
	if len(points) < 2 {
		return 0
	}

	totalAscent := 0.0
	climb := 0.0
	prevEle := points[0].Elevation

	for i := 1; i < len(points); i++ {
		currEle := points[i].Elevation
		diff := currEle - prevEle

		if diff > 0 {
			climb += diff
		} else {

			if climb >= threshold {
				totalAscent += climb
			}
			climb = 0
		}
		prevEle = currEle
	}
	if climb >= threshold {
		totalAscent += climb
	}

	return totalAscent
}

func applyLatLonSmoothing(points []TrackPoint, windowSize int) []TrackPoint {
	if windowSize < 1 || len(points) == 0 {
		return points
	}
	if windowSize%2 == 0 {
		windowSize++
	}

	smoothed := make([]TrackPoint, len(points))
	half := windowSize / 2

	for i := range points {
		sumLat, sumLon := 0.0, 0.0
		count := 0

		start := i - half
		if start < 0 {
			start = 0
		}
		end := i + half
		if end >= len(points) {
			end = len(points) - 1
		}

		for j := start; j <= end; j++ {
			sumLat += points[j].Latitude
			sumLon += points[j].Longitude
			count++
		}

		smoothed[i] = points[i]
		smoothed[i].Latitude = sumLat / float64(count)
		smoothed[i].Longitude = sumLon / float64(count)
	}
	return smoothed
}

func calculateGOTAscent(n float64) int {
	val := n / 100.0
	return int(math.Round(val))
}

func calculateGOTDistance(n float64) int {
	return int(math.Round(n))
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run your_program_name.go <gpx_file_path>")
		os.Exit(1)
	}

	filePath := os.Args[1]

	gpxData, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading GPX file: %v\n", err)
		return
	}

	var gpx GPX
	err = xml.Unmarshal(gpxData, &gpx)
	if err != nil {
		fmt.Printf("Error unmarshaling GPX XML: %v\n", err)
		return
	}

	fmt.Printf("Successfully parsed GPX file. Found %d tracks.\n", len(gpx.Tracks))

	totalDistanceKm := 0.0
	totalAscentMeters := 0.0

	const ascentThreshold = 1.5
	const movingAverageWindowSize = 3

	for _, track := range gpx.Tracks {
		fmt.Printf("Track Name: %s\n", track.Name)
		for _, segment := range track.TrackSegs {
			smoothed := applyMovingAverage(segment.TrackPoints, movingAverageWindowSize)
			smoothed = applyLatLonSmoothing(smoothed, movingAverageWindowSize)

			for i := 1; i < len(smoothed); i++ {
				totalDistanceKm += haversineDistance2D(smoothed[i-1], smoothed[i])
			}
			totalAscentMeters += calculateCumulativeAscent(smoothed, ascentThreshold)
		}
	}

	fmt.Printf("\n--- Results ---\n")
	fmt.Printf("Total Distance: %.2f km\n", totalDistanceKm)
	fmt.Printf("Total Ascent: %.2f meters (smoothed with  %.1fm threshold)\n",
		totalAscentMeters, ascentThreshold)
	fmt.Printf("\n--- GOT points ---\n")
	fmt.Printf("Distance points: %d pkt\n", calculateGOTDistance(totalDistanceKm))
	fmt.Printf("Ascent points: %d pkt\n", calculateGOTAscent(totalAscentMeters))

}
