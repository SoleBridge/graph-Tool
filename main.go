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
)

type Tool int

const (
	ToolAddVertex Tool = iota
	ToolAddEdge
	ToolDeleteVertex
	ToolDeleteEdge
	ToolMoveVertex
	ToolColorVertex
)

var toolNames = []string{
	"Add Vertex",
	"Add Edge",
	"Delete Vertex",
	"Delete Edge",
	"Move Vertex",
	"Color Vertex",
}

type Vertex struct {
	X, Y  float64
	Label string
	Color color.RGBA
}

type Graph struct {
	Vertices   []Vertex
	AdjMatrix  [][]int
	IsDirected bool
}

type App struct {
	Graph        *Graph
	Selected     *int // Index of the selected vertex
	Tool         Tool // Current active tool
	EdgeStart    *int // Start vertex for adding an edge
	MovingVertex *int // Index of the vertex being moved
	LastClickTime time.Time
}

// AddVertex adds a new vertex to the graph.
func (g *Graph) AddVertex(x, y float64, label string, clr color.RGBA) {
	g.Vertices = append(g.Vertices, Vertex{X: x, Y: y, Label: label, Color: clr})
	for i := range g.AdjMatrix {
		g.AdjMatrix[i] = append(g.AdjMatrix[i], 0)
	}
	g.AdjMatrix = append(g.AdjMatrix, make([]int, len(g.Vertices)))
}

// DeleteVertex removes a vertex and its associated edges.
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

// DeleteEdge removes an edge between two vertices.
func (g *Graph) DeleteEdge(v1, v2 int) {
	if v1 < 0 || v2 < 0 || v1 >= len(g.Vertices) || v2 >= len(g.Vertices) {
		return
	}
	g.AdjMatrix[v1][v2] = 0
	if !g.IsDirected {
		g.AdjMatrix[v2][v1] = 0
	}
}

// NewApp initializes the application.
func NewApp() *App {
	return &App{
		Graph: &Graph{
			Vertices:   []Vertex{},
			AdjMatrix:  [][]int{},
			IsDirected: false,
		},
		Tool: ToolAddVertex,
	}
}

// HandleMouseInput processes mouse interactions.
func (app *App) HandleMouseInput() {
	x, y := ebiten.CursorPosition()
	mx, my := float64(x), float64(y)

	if my < 40 {
		// Toolbar interaction (simple button clicks)
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			// Map toolbar region to tools (assuming 100px wide buttons)
			toolIndex := int(mx) / 100
			if toolIndex >= 0 && toolIndex < len(toolNames) {
				app.Tool = Tool(toolIndex)
			}
		}
		return
	}

	// Handle other mouse clicks based on current tool
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		switch app.Tool {
		case ToolAddVertex:
			app.Graph.AddVertex(mx, my, fmt.Sprintf("V%d", len(app.Graph.Vertices)+1), color.RGBA{255, 0, 0, 255})
		case ToolAddEdge:
			for i, v := range app.Graph.Vertices {
				if math.Hypot(v.X-mx, v.Y-my) < 15 {
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
			for i, v1 := range app.Graph.Vertices {
				if math.Hypot(v1.X-mx, v1.Y-my) < 15 {
					for j := range app.Graph.Vertices {
						if i != j && app.Graph.AdjMatrix[i][j] > 0 {
							app.Graph.DeleteEdge(i, j)
							return
						}
					}
				}
			}
		case ToolColorVertex:
			for i, v := range app.Graph.Vertices {
				if math.Hypot(v.X-mx, v.Y-my) < 15 {
					app.Graph.Vertices[i].Color = color.RGBA{0, 255, 0, 255} // Green color
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

// AddEdge adds an edge between two vertices, allowing parallel edges and loops.
func (g *Graph) AddEdge(v1, v2 int) {
	if v1 < 0 || v2 < 0 || v1 >= len(g.Vertices) || v2 >= len(g.Vertices) {
		return
	}
	g.AdjMatrix[v1][v2]++
	if !g.IsDirected && v1 != v2 {
		g.AdjMatrix[v2][v1]++
	}
}

func DrawLinearBézierEdge(screen *ebiten.Image, x1, y1, x2, y2, cx, cy float64, clr color.RGBA) {
	for t := 0.0; t <= 1.0; t += 0.001 {
		x := (1-t)*(1-t)*x1 + 2*(1-t)*t*cx + t*t*x2
		y := (1-t)*(1-t)*y1 + 2*(1-t)*t*cy + t*t*y2
		ebitenutil.DrawRect(screen, x, y, 1, 1, clr)
	}
}

func DrawQuadraticBézierEdge(screen *ebiten.Image, x1, y1, x2, y2, xc1, yc1, xc2, yc2 float64, clr color.RGBA) {
    for t := 0.0; t <= 1.0; t += 0.001 {
        // Cubic Bézier curve equation
        x := (1-t)*(1-t)*(1-t)*x1 + 3*(1-t)*(1-t)*t*xc1 + 3*(1-t)*t*t*xc2 + t*t*t*x2
        y := (1-t)*(1-t)*(1-t)*y1 + 3*(1-t)*(1-t)*t*yc1 + 3*(1-t)*t*t*yc2 + t*t*t*y2
        ebitenutil.DrawRect(screen, x, y, 1, 1, clr)
    }
}

func (g *Graph) DrawEdges(screen *ebiten.Image) {
	edgeColor := color.RGBA{255, 0, 0, 255} // Red

	for i, v1 := range g.Vertices {
		for j, v2 := range g.Vertices {
			count := g.AdjMatrix[i][j]
			if count > 0 {
				if i == j {
					// Loop: Adjust control points for a more realistic curve
					DrawQuadraticBézierEdge(screen, v1.X, v1.Y, v1.X, v1.Y, v1.X-45, v1.Y-45, v1.X+45, v1.Y-45, edgeColor)
				} else if count == 1 {
					// Single edge: draw a straight line
					ebitenutil.DrawLine(screen, v1.X, v1.Y, v2.X, v2.Y, edgeColor)
				} else {
					// Parallel edges: draw Bézier curves
					for k := 0; k < count; k++ {
						offset := float64(15 * (k - count/2)) // Offset for parallel edges
						cx, cy := (v1.X+v2.X)/2+offset, (v1.Y+v2.Y)/2-offset
						DrawLinearBézierEdge(screen, v1.X, v1.Y, v2.X, v2.Y, cx, cy, edgeColor)
					}
				}
			}
		}
	}
}


// Draw the toolbar and the selected tool indicator
func (app *App) Draw(screen *ebiten.Image) {
	// Draw toolbar (buttons)
	for i, toolName := range toolNames {
		toolColor := color.RGBA{200, 200, 200, 255}
		if app.Tool == Tool(i) {
			toolColor = color.RGBA{100, 100, 255, 255} // Highlight selected tool
		}
		// Draw button background
		ebitenutil.DrawRect(screen, float64(i*100), 0, 100, 40, toolColor)
		// Draw tool name
		ebitenutil.DebugPrintAt(screen, toolName, i*100+5, 10)
	}

	// Draw edges
	app.Graph.DrawEdges(screen)

	// Draw vertices
	for _, v := range app.Graph.Vertices {
		ebitenutil.DrawCircle(screen, v.X, v.Y, 15, v.Color)
		ebitenutil.DebugPrintAt(screen, v.Label, int(v.X)-10, int(v.Y)-5)
	}
}

// Update processes game logic.
func (app *App) Update() error {
	app.HandleMouseInput()
	return nil
}

// Layout sets the screen size.
func (app *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 800, 600
}

func main() {
	app := NewApp()
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("Graph Tool")
	if err := ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}
}
