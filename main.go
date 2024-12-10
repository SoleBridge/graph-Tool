package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Tool types enum and label strings:

type Tool int

const (
	ToolAddVertex Tool = iota // used for auto-increment
	ToolAddEdge
	ToolDeleteVertex
	ToolDeleteEdge
	ToolMoveVertex
	ToolColorVertex
	ToolNameVertex
	ToolPrintInfo
)

var toolNames = []string{
	"Add Vertex",
	"Add Edge",
	"Delete Vertex",
	"Delete Edge",
	"Move Vertex",
	"Color Vertex",
	"Name Vertex",
	"Print Info",
}

// Vertex and graph info:

// Edges stored via adjacency matrix.
// Vertex drawing info in it's own struct.
// Vertices are tracked by index, in adjacency matrix and vertex slice.

type Vertex struct {
	X, Y  float64
	Label string
	Color color.RGBA
}

type Graph struct {
	Vertices  []Vertex
	AdjMatrix [][]int
}

// Adds a vertex to the graph.
func (g *Graph) AddVertex(x, y float64, label string, clr color.RGBA) {
	g.Vertices = append(g.Vertices, Vertex{X: x, Y: y, Label: label, Color: clr})
	// Expand adjacency matrix:
	for i := range g.AdjMatrix {
		g.AdjMatrix[i] = append(g.AdjMatrix[i], 0)
	}
	g.AdjMatrix = append(g.AdjMatrix, make([]int, len(g.Vertices)))
}

// Removes a vertex (and its edges) from the graph.
// Uses some fun slice indexing.
func (g *Graph) DeleteVertex(index int) {
	if index < 0 || index >= len(g.Vertices) {
		return
	}
	g.Vertices = append(g.Vertices[:index], g.Vertices[index+1:]...)
	g.AdjMatrix = append(g.AdjMatrix[:index], g.AdjMatrix[index+1:]...)
	for i := range g.AdjMatrix {
		g.AdjMatrix[i] = append(g.AdjMatrix[i][:index], g.AdjMatrix[i][index+1:]...)
	}
}

// Removes an edge.
func (g *Graph) DeleteEdge(v1, v2 int) {
	if v1 < 0 || v2 < 0 || v1 >= len(g.Vertices) || v2 >= len(g.Vertices) {
		return
	}

	if v1 == v2 { // Loop
		if g.AdjMatrix[v1][v1] > 0 {
			g.AdjMatrix[v1][v1]--
		}
	} else { // Non-loop
		g.AdjMatrix[v1][v2]--
		g.AdjMatrix[v2][v1]--
	}
}

// Adds an edge between two vertices (allows parallel edges and loops - using Brezier curves).
func (g *Graph) AddEdge(v1, v2 int) {
	if v1 < 0 || v2 < 0 || v1 >= len(g.Vertices) || v2 >= len(g.Vertices) {
		return
	}

	g.AdjMatrix[v1][v2]++
	if v1 != v2 { // Only count loops once
		g.AdjMatrix[v2][v1]++
	}
}

// App struct to hold application info

type App struct {
	Graph         *Graph    // Graph
	Selected      *int      // Selected vertex (index)
	Tool          Tool      // Selected tool
	EdgeStart     *int      // Start vertex for adding an edge
	MovingVertex  *int      // Index of the vertex being moved
	LastClickTime time.Time // For vertex adding delay
}

// Initializes the app.
func NewApp() *App {
	return &App{
		Graph: &Graph{
			Vertices:  []Vertex{},
			AdjMatrix: [][]int{},
		},
		Tool: ToolAddVertex,
	}
}

// Processes mouse interactions.
func (app *App) HandleMouseInput() {
	x, y := ebiten.CursorPosition()
	mx, my := float64(x), float64(y)

	// Handle other mouse clicks based on current tool
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// Toolbar zone (assumes 100px wide buttons)
		if my < 40 {
			toolIndex := int(mx) / 100
			if toolIndex >= 0 && toolIndex < len(toolNames) {
				// Change selected tool if it's not print info
				old_tool := app.Tool
				app.Tool = Tool(toolIndex)
				if app.Tool == ToolPrintInfo {
					app.printGraphInfo()
					app.Tool = old_tool
				}
			}
			return
		}

		switch app.Tool {
		case ToolAddVertex:
			app.Graph.AddVertex(mx, my, fmt.Sprintf("V%d", len(app.Graph.Vertices)+1), color.RGBA{255, 0, 0, 255})
		case ToolAddEdge:
			for i, v := range app.Graph.Vertices { // Look thru vertices
				if math.Hypot(v.X-mx, v.Y-my) < 15 { // To find one near mouse
					if app.EdgeStart == nil {
						app.EdgeStart = &i
					} else {
						app.Graph.AddEdge(*app.EdgeStart, i)
						app.EdgeStart = nil
					}
					return
				}
			}
		case ToolDeleteVertex:
			for i, v := range app.Graph.Vertices {
				if math.Hypot(v.X-mx, v.Y-my) < 15 {
					app.Graph.DeleteVertex(i)
					return
				}
			}
		case ToolDeleteEdge:
			// This whole thing could probably be better than O(n^4)
			for i, v1 := range app.Graph.Vertices {
				for j, v2 := range app.Graph.Vertices {
					if i == j {
						continue // Skip loops
					}

					if app.Graph.AdjMatrix[i][j] > 0 {
						// Check line:
						dist := pointToLineDistance(mx, my, v1.X, v1.Y, v2.X, v2.Y)
						if dist < 10 {
							app.Graph.DeleteEdge(i, j)
							return
						}

						// Check parallel edges:
						count := app.Graph.AdjMatrix[i][j]
						for k := 0; k < count; k++ {
							offset := float64(15 * (k - count/2))
							cx, cy := (v1.X+v2.X)/2+offset, (v1.Y+v2.Y)/2-offset
							dist := pointToBezierDistance(mx, my, v1.X, v1.Y, v2.X, v2.Y, cx, cy)
							if dist < 10 {
								app.Graph.DeleteEdge(i, j)
								return
							}
						}
					}
				}
			}

			// Handle loops
			for i, v1 := range app.Graph.Vertices {
				if app.Graph.AdjMatrix[i][i] > 0 {
					count := app.Graph.AdjMatrix[i][i]
					for k := 0; k < count; k++ {
						angleOffset := float64(k) * (2 * math.Pi / float64(count))
						angleLeft := angleOffset - math.Pi/10
						angleRight := angleOffset + math.Pi/10
						cxLeft := v1.X + 60*math.Cos(angleLeft)
						cyLeft := v1.Y + 60*math.Sin(angleLeft)
						cxRight := v1.X + 60*math.Cos(angleRight)
						cyRight := v1.Y + 60*math.Sin(angleRight)
						dist := pointToQuadraticBezierDistance(mx, my, v1.X, v1.Y, v1.X, v1.Y, cxLeft, cyLeft, cxRight, cyRight)
						if dist < 10 {
							app.Graph.DeleteEdge(i, i)
							return
						}
					}
				}
			}

		case ToolColorVertex:
			for i, v := range app.Graph.Vertices {
				if math.Hypot(v.X-mx, v.Y-my) < 15 {
					app.Graph.Vertices[i].Color = color.RGBA{0, 255, 0, 255}
					return
				}
			}
		case ToolNameVertex:
			for i, v := range app.Graph.Vertices {
				if math.Hypot(v.X-mx, v.Y-my) < 15 {
					app.Selected = &i
					fmt.Printf("Name V%d: ", i)
					var newName string
					fmt.Scanln(&newName)
					app.Graph.Vertices[i].Label = newName
					return
				}
			}
		}
	}

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if app.Tool == ToolMoveVertex {
			for i, v := range app.Graph.Vertices {
				if math.Hypot(v.X-mx, v.Y-my) < 15 {
					app.MovingVertex = &i
					break
				}
			}
			if app.MovingVertex != nil {
				v := &app.Graph.Vertices[*app.MovingVertex]
				v.X, v.Y = mx, my
			}
		}
	}

	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		app.MovingVertex = nil
	}
}

// Drawing functions:

// Draws a Bézier curve from (x1,y1) to (x2,y2) with control point (cx,cy).
func DrawLinearBézierEdge(screen *ebiten.Image, x1, y1, x2, y2, cx, cy float64, clr color.RGBA) {
	for t := 0.0; t <= 1.0; t += 0.001 {
		x := (1-t)*(1-t)*x1 + 2*(1-t)*t*cx + t*t*x2
		y := (1-t)*(1-t)*y1 + 2*(1-t)*t*cy + t*t*y2
		vector.DrawFilledRect(screen, float32(x), float32(y), 1, 1, clr, true)
	}
}

// Draws a Bézier curve from (x1,y1) to (x2,y2) with control points (cx1,cy1) and (cx2,cy2).
func DrawQuadraticBézierEdge(screen *ebiten.Image, x1, y1, x2, y2, xc1, yc1, xc2, yc2 float64, clr color.RGBA) {
	for t := 0.0; t <= 1.0; t += 0.001 {
		x := (1-t)*(1-t)*(1-t)*x1 + 3*(1-t)*(1-t)*t*xc1 + 3*(1-t)*t*t*xc2 + t*t*t*x2
		y := (1-t)*(1-t)*(1-t)*y1 + 3*(1-t)*(1-t)*t*yc1 + 3*(1-t)*t*t*yc2 + t*t*t*y2
		vector.DrawFilledRect(screen, float32(x), float32(y), 1, 1, clr, true)
	}
}

// Draws all edges of the graph.
func (g *Graph) DrawEdges(screen *ebiten.Image) {
	edgeColor := color.RGBA{255, 0, 0, 255}

	for i, v1 := range g.Vertices {
		for j, v2 := range g.Vertices {
			count := g.AdjMatrix[i][j]
			if count > 0 {
				if i == j { // Loop: Bézier curve
					DrawLoopEdge(screen, v1.X, v1.Y, count, edgeColor)
				} else if count == 1 { // Single edge: straight line
					vector.StrokeLine(screen, float32(v1.X), float32(v1.Y), float32(v2.X), float32(v2.Y), 3.0, edgeColor, true)
				} else { // Parallel edges: Bézier curves
					for k := 0; k < count; k++ {
						offset := float64(20 * (k - count/2)) // Offset for parallel edges
						cx, cy := (v1.X+v2.X)/2+offset, (v1.Y+v2.Y)/2-offset
						DrawLinearBézierEdge(screen, v1.X, v1.Y, v2.X, v2.Y, cx, cy, edgeColor)
					}
				}
			}
		}
	}
}

// Application functions.

// Displays graph information.
//
//	Adjacency matrix.
//	Number of edges and vertices.
//	Degree of each vertex.
func (app *App) printGraphInfo() {
	numVertices := len(app.Graph.Vertices)
	numEdges := 0
	degrees := make([]int, numVertices)

	// Print header:
	fmt.Println("\nAdjacency Matrix:")
	fmt.Printf("          ")
	for _, vertex := range app.Graph.Vertices {
		fmt.Printf("%-10s", vertex.Label)
	}
	fmt.Println()

	// Print adjacency matrix:
	for i := range app.Graph.AdjMatrix {
		fmt.Printf("%-10s", app.Graph.Vertices[i].Label)
		for j := range app.Graph.AdjMatrix[i] {
			fmt.Printf("%-10d", app.Graph.AdjMatrix[i][j])
			if app.Graph.AdjMatrix[i][j] > 0 {
				degrees[i] += app.Graph.AdjMatrix[i][j]
			}
			numEdges += app.Graph.AdjMatrix[i][j]
		}
		fmt.Println()
	}

	// Print other graph information:
	fmt.Printf("\n# vertices: %d\n", numVertices)
	fmt.Printf("# edges: %d\n", numEdges)
	for i, degree := range degrees {
		fmt.Printf("deg(V%d \"%s\"): %d\n", i, app.Graph.Vertices[i].Label, degree)
	}
}

// Draws loop(s) evenly spaced around the vertex.
func DrawLoopEdge(screen *ebiten.Image, x, y float64, count int, clr color.RGBA) {
	for i := 0; i < count; i++ {
		// Find angle to middle of loop
		angleOffset := float64(i) * (2 * math.Pi / float64(count))
		// Find angle to either side (for 2 Brezier control points)
		angleLeft := angleOffset - math.Pi/10
		angleRight := angleOffset + math.Pi/10
		// Find control points
		cxLeft := x + 60*math.Cos(angleLeft)
		cyLeft := y + 60*math.Sin(angleLeft)
		cxRight := x + 60*math.Cos(angleRight)
		cyRight := y + 60*math.Sin(angleRight)
		// Draw Brezier
		DrawQuadraticBézierEdge(screen, x, y, x, y, cxLeft, cyLeft, cxRight, cyRight, clr)
	}
}

// Draws the application.
func (app *App) Draw(screen *ebiten.Image) {
	// Draw toolbar
	for i, toolName := range toolNames {
		toolColor := color.RGBA{200, 200, 200, 255}
		if app.Tool == Tool(i) {
			toolColor = color.RGBA{100, 100, 255, 255} // Highlight selected tool
		}
		vector.DrawFilledRect(screen, float32(i*100), 0, 100, 40, toolColor, true)
		ebitenutil.DebugPrintAt(screen, toolName, i*100+5, 10)
	}

	// Draw edges
	app.Graph.DrawEdges(screen)

	// Draw vertices
	for _, v := range app.Graph.Vertices {
		vector.DrawFilledCircle(screen, float32(v.X), float32(v.Y), 15, v.Color, true)
		ebitenutil.DebugPrintAt(screen, v.Label, int(v.X)-10, int(v.Y)-5)
	}
}

// Computes next frame.
func (app *App) Update() error {
	app.HandleMouseInput()
	return nil
}

// Sets the screen size.
func (app *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 800, 600
}

// Helper functions:

// Calculate the distance from a point (mx, my) to a line segment (x1, y1) -> (x2, y2).
func pointToLineDistance(mx, my, x1, y1, x2, y2 float64) float64 {
	lineLength := math.Hypot(x2-x1, y2-y1)
	if lineLength == 0 {
		// Simple case: line is a point
		return math.Hypot(mx-x1, my-y1)
	}
	t := ((mx-x1)*(x2-x1) + (my-y1)*(y2-y1)) / (lineLength * lineLength)
	t = math.Max(0, math.Min(1, t))
	closestX := x1 + t*(x2-x1)
	closestY := y1 + t*(y2-y1)
	return math.Hypot(mx-closestX, my-closestY)
}

// Calculate the distance from a point (mx, my) to a Bézier curve (x1, y1) -> (x2, y2) with control point (cx, cy).
// Done by sample along the Bézier curve and find the closest point.
func pointToBezierDistance(mx, my, x1, y1, x2, y2, cx, cy float64) float64 {
	closestDist := math.Inf(1)
	for t := 0.0; t <= 1.0; t += 0.001 {
		x := (1-t)*(1-t)*x1 + 2*(1-t)*t*cx + t*t*x2
		y := (1-t)*(1-t)*y1 + 2*(1-t)*t*cy + t*t*y2
		dist := math.Hypot(mx-x, my-y)
		if dist < closestDist {
			closestDist = dist
		}
	}
	return closestDist
}

// Calculate the distance from a point (mx, my) to a Bézier curve (x1, y1) -> (x2, y2) with control point (cx, cy).
// Done by sampling along the Bézier curve and finding the closest point.
func pointToLinearBezierDistance(mx, my, x1, y1, x2, y2, cx, cy float64) float64 {
	closestDist := math.Inf(1)
	for t := 0.0; t <= 1.0; t += 0.001 {
		x := (1-t)*(1-t)*x1 + 2*(1-t)*t*cx + t*t*x2
		y := (1-t)*(1-t)*y1 + 2*(1-t)*t*cy + t*t*y2
		dist := math.Hypot(mx-x, my-y)
		if dist < closestDist {
			closestDist = dist
		}
	}
	return closestDist
}

// Calculate the distance from a point (mx, my) to a Bézier curve (x1, y1) -> (x2, y2) with control points (cx1, cy1) and (cx2, cy2).
// Done by sampling along the Bézier curve and finding the closest point.
func pointToQuadraticBezierDistance(mx, my, x1, y1, x2, y2, cx1, cy1, cx2, cy2 float64) float64 {
	closestDist := math.Inf(1)
	for t := 0.0; t <= 1.0; t += 0.001 {
		x := (1-t)*(1-t)*(1-t)*x1 + 3*(1-t)*(1-t)*t*cx1 + 3*(1-t)*t*t*cx2 + t*t*t*x2
		y := (1-t)*(1-t)*(1-t)*y1 + 3*(1-t)*(1-t)*t*cy1 + 3*(1-t)*t*t*cy2 + t*t*t*y2
		dist := math.Hypot(mx-x, my-y)
		if dist < closestDist {
			closestDist = dist
		}
	}
	return closestDist
}

// Entry point.

func main() {
	app := NewApp()
	ebiten.SetWindowSize(1920, 1080)
	ebiten.SetWindowTitle("Graph Tool")
	if err := ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}
}
