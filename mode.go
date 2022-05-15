package main

import (
	"image"
	"math"
	"time"

	"deedles.dev/kawa/ui"
	"deedles.dev/wlr"
	"golang.org/x/exp/slices"
)

type InputMode interface{}

type inputModeNormal struct {
	inView    bool
	prevEdges wlr.Edges
}

func (server *Server) startNormal() {
	server.setCursor("left_ptr")
	server.inputMode = &inputModeNormal{}
}

func (m *inputModeNormal) CursorMoved(server *Server, t time.Time) {
	x, y := server.cursor.X(), server.cursor.Y()

	view, edges, surface, sx, sy := server.viewAt(nil, x, y)
	if (edges != m.prevEdges) && !server.isViewTiled(view) {
		server.setCursor(edgeCursors[edges])
		m.prevEdges = edges
	}
	if (view == nil) && m.inView {
		server.setCursor("left_ptr")
	}
	m.inView = view != nil
	if !surface.Valid() {
		server.seat.PointerNotifyClearFocus()
		return
	}

	server.seat.PointerNotifyEnter(surface, sx, sy)
	server.seat.PointerNotifyMotion(t, sx, sy)
}

func (m *inputModeNormal) CursorButtonPressed(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	view, edges, surface, _, _ := server.viewAt(nil, server.cursor.X(), server.cursor.Y())
	if view == nil {
		if b == wlr.BtnRight {
			server.startMenu(server.mainMenu)
		}
		return
	}

	server.focusView(view, surface)

	switch edges {
	case wlr.EdgeNone:
		server.seat.PointerNotifyButton(t, b, wlr.ButtonPressed)
	default:
		switch b {
		case wlr.BtnLeft:
			if !server.isViewTiled(view) {
				server.startBorderResize(view, edges)
			}
		case wlr.BtnRight:
			server.startMove(view)
		}
	}
}

func (m *inputModeNormal) CursorButtonReleased(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	server.seat.PointerNotifyButton(t, b, wlr.ButtonReleased)
}

func (m *inputModeNormal) RequestCursor(server *Server, s wlr.Surface, x, y int) {
	server.cursor.SetSurface(s, int32(x), int32(y))
}

type inputModeMove struct {
	view *View
	off  image.Point
}

func (server *Server) startMove(view *View) {
	server.setCursor("grabbing")
	server.focusView(view, view.Surface())

	x, y := server.cursor.X(), server.cursor.Y()
	server.inputMode = &inputModeMove{
		view: view,
		off:  image.Pt(int(x)-view.X, int(y)-view.Y),
	}
}

func (m *inputModeMove) CursorMoved(server *Server, t time.Time) {
	x, y := server.cursor.X(), server.cursor.Y()

	if !server.isViewTiled(m.view) {
		p := image.Pt(int(x), int(y)).Sub(m.off)
		server.moveViewTo(nil, m.view, p.X, p.Y)
		return
	}

	i, _, _, _, _ := server.viewIndexAt(nil, server.tiled, x, y)
	if i >= 0 {
		vi := slices.Index(server.tiled, m.view)
		server.tiled[i], server.tiled[vi] = server.tiled[vi], server.tiled[i]
		server.layoutTiles(nil)
	}
}

func (m *inputModeMove) CursorButtonReleased(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	server.startNormal()
}

func (m *inputModeMove) TargetView() *View {
	return m.view
}

type inputModeBorderResize struct {
	view  *View
	edges wlr.Edges
	cur   image.Rectangle
}

func (server *Server) startBorderResize(view *View, edges wlr.Edges) {
	vb := server.viewBounds(nil, view)
	server.startBorderResizeFrom(view, edges, vb)
}

func (server *Server) startBorderResizeFrom(view *View, edges wlr.Edges, from image.Rectangle) {
	view.SetResizing(true)
	server.focusView(view, view.Surface())
	server.inputMode = &inputModeBorderResize{
		view:  view,
		edges: edges,
		cur:   from,
	}
}

func (m *inputModeBorderResize) CursorMoved(server *Server, t time.Time) {
	x, y := server.cursor.X(), server.cursor.Y()
	ox, oy := int(x), int(y)

	r := m.cur
	if m.edges&wlr.EdgeTop != 0 {
		r.Min.Y = oy
		if r.Dy() < ui.MinHeight {
			r.Min.Y = r.Max.Y - ui.MinHeight
		}
	}
	if m.edges&wlr.EdgeBottom != 0 {
		r.Max.Y = oy
		if r.Dy() < ui.MinHeight {
			r.Max.Y = r.Min.Y + ui.MinHeight
		}
	}
	if m.edges&wlr.EdgeLeft != 0 {
		r.Min.X = ox
		if r.Dx() < ui.MinWidth {
			r.Min.X = r.Max.X - ui.MinWidth
		}
	}
	if m.edges&wlr.EdgeRight != 0 {
		r.Max.X = ox
		if r.Dx() < ui.MinWidth {
			r.Max.X = r.Min.X + ui.MinWidth
		}
	}

	if ox < r.Min.X {
		m.edges |= wlr.EdgeLeft
		m.edges &^= wlr.EdgeRight
		server.setCursor(edgeCursors[m.edges])
	}
	if ox > r.Max.X {
		m.edges |= wlr.EdgeRight
		m.edges &^= wlr.EdgeLeft
		server.setCursor(edgeCursors[m.edges])
	}
	if oy < r.Min.Y {
		m.edges |= wlr.EdgeTop
		m.edges &^= wlr.EdgeBottom
		server.setCursor(edgeCursors[m.edges])
	}
	if oy > r.Max.Y {
		m.edges |= wlr.EdgeBottom
		m.edges &^= wlr.EdgeTop
		server.setCursor(edgeCursors[m.edges])
	}

	m.cur = r
	server.resizeViewTo(nil, m.view, r.Canon())
}

func (m *inputModeBorderResize) CursorButtonReleased(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	m.view.SetResizing(false)
	server.startNormal()
}

func (m *inputModeBorderResize) TargetView() *View {
	return m.view
}

type inputModeMenu struct {
	m   *ui.Menu
	p   image.Point
	sel int
}

func (server *Server) startMenu(m *ui.Menu) {
	x, y := server.cursor.X(), server.cursor.Y()
	out := server.outputAt(x, y)
	ob := box(0, 0, out.Output.Width(), out.Output.Height())

	mb := m.Bounds().Add(m.StartOffset()).Add(image.Pt(int(x), int(y)))

	if i := mb.Intersect(ob); mb != i {
		before := mb
		mb = mb.Sub(before.Min.Sub(i.Min))
		mb = mb.Sub(before.Max.Sub(i.Max))
	}

	mode := inputModeMenu{
		m: m,
		p: mb.Min,
	}
	mode.CursorMoved(server, time.Now())
	server.inputMode = &mode
}

func (m *inputModeMenu) CursorMoved(server *Server, t time.Time) {
	cx, cy := server.cursor.X(), server.cursor.Y()

	p := image.Pt(int(cx), int(cy))
	r := m.m.Bounds().Add(m.p)

	m.sel = -1
	if p.In(r) {
		m.sel = (p.Y - r.Min.Y) / int(fontOptions.Size+ui.WindowBorder*2)
	}
}

func (m *inputModeMenu) CursorButtonReleased(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	if b != wlr.BtnRight {
		return
	}

	server.startNormal()
	m.m.Select(m.sel)
}

func (m *inputModeMenu) Frame(server *Server, out *Output, t time.Time) {
	server.renderMenu(out, m.m, float64(m.p.X), float64(m.p.Y), m.sel)
}

type inputModeSelectView struct {
	startBtn wlr.CursorButton
	then     func(*View)
}

func (server *Server) startSelectView(b wlr.CursorButton, then func(*View)) {
	server.setCursor("hand1")
	server.inputMode = &inputModeSelectView{
		startBtn: b,
		then:     then,
	}
}

func (m *inputModeSelectView) CursorButtonPressed(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	if b != m.startBtn {
		server.startNormal()
		return
	}

	x, y := server.cursor.X(), server.cursor.Y()
	view, _, _, _, _ := server.viewAt(nil, x, y)
	if view != nil {
		m.then(view)
		return
	}
	server.startNormal()
}

type inputModeResize struct {
	view     *View
	sx, sy   float64
	resizing bool
}

func (server *Server) startResize(view *View) {
	server.setCursor("top_left_corner")
	server.inputMode = &inputModeResize{
		view: view,
	}
}

func (m *inputModeResize) CursorMoved(server *Server, t time.Time) {
	if !m.resizing {
		return
	}

	x, y := server.cursor.X(), server.cursor.Y()
	if math.Abs(x-m.sx) < ui.MinWidth {
		return
	}
	if math.Abs(y-m.sy) < ui.MinHeight {
		return
	}

	r := image.Rect(
		int(m.sx),
		int(m.sy),
		int(x),
		int(y),
	)
	if server.isViewTiled(m.view) {
		server.untileView(m.view, false)
	}
	server.startBorderResizeFrom(m.view, wlr.EdgeNone, r)
}

func (m *inputModeResize) CursorButtonPressed(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	if b != wlr.BtnRight {
		server.startNormal()
		return
	}

	m.sx, m.sy = server.cursor.X(), server.cursor.Y()
	m.resizing = true
}

func (m *inputModeResize) CursorButtonReleased(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	if !m.resizing {
		return
	}

	x, y := server.cursor.X(), server.cursor.Y()
	r := image.Rect(int(m.sx), int(m.sy), int(x), int(y))
	if (r.Dx() >= ui.MinWidth) && (r.Dy() >= ui.MinHeight) {
		server.resizeViewTo(nil, m.view, r)
	}
	server.startNormal()
}

func (m *inputModeResize) Frame(server *Server, out *Output, t time.Time) {
	if !m.resizing {
		return
	}

	x, y := server.cursor.X(), server.cursor.Y()
	r := image.Rect(
		int(m.sx),
		int(m.sy),
		int(x),
		int(y),
	).Canon()
	server.renderSelectionBox(out, r, t)
}

func (m *inputModeResize) TargetView() *View {
	return m.view
}

type inputModeNew struct {
	n        image.Rectangle
	dragging bool
	started  bool
}

func (server *Server) startNew() {
	server.setCursor("top_left_corner")
	server.inputMode = &inputModeNew{}
}

func (m *inputModeNew) CursorMoved(server *Server, t time.Time) {
	if !m.dragging {
		return
	}

	x, y := server.cursor.X(), server.cursor.Y()
	if math.Abs(x-float64(m.n.Min.X)) < ui.MinWidth {
		return
	}
	if math.Abs(y-float64(m.n.Min.Y)) < ui.MinHeight {
		return
	}

	m.n.Max.X = int(x)
	m.n.Max.Y = int(y)

	if !m.started {
		server.exec(&m.n)
		m.started = true
	}
}

func (m *inputModeNew) CursorButtonPressed(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	if b != wlr.BtnRight {
		server.startNormal()
		return
	}

	m.n.Min.X, m.n.Min.Y = int(server.cursor.X()), int(server.cursor.Y())
	m.dragging = true
}

func (m *inputModeNew) CursorButtonReleased(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	if !m.dragging {
		return
	}

	server.startNormal()
}

func (m *inputModeNew) Frame(server *Server, out *Output, t time.Time) {
	if !m.dragging || m.started {
		return
	}

	x, y := server.cursor.X(), server.cursor.Y()
	r := image.Rect(
		int(m.n.Min.X),
		int(m.n.Min.Y),
		int(x),
		int(y),
	)
	server.renderSelectionBox(out, r, t)
}
