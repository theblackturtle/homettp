package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/homettp/homettp/internal/models"
)

func (app *App) callIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "POST" {
		app.methodNotAllowed(w, []string{"GET", "POST"})

		return
	}

	command, err := app.getCommandFromRequest(r, "id")
	if err != nil {
		app.notFound(w)

		return
	}

	if command.Token != r.URL.Query().Get("token") {
		app.notFound(w)

		return
	}

	call := models.NewCall(command)
	call.Payload = r.URL.Query().Get("payload")

	err = app.callRepository.Create(call)
	if err != nil {
		app.serverError(w, err)

		return
	}

	select {
	case app.queue <- call.Id:
		w.Header().Set("Content-Type", "application/json")

		err = json.NewEncoder(w).Encode(call)
		if err != nil {
			app.serverError(w, err)
		}
	default:
		app.clientError(w, 503)
	}
}

func (app *App) callHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		app.methodNotAllowed(w, []string{"GET"})

		return
	}

	command, err := app.getCommandFromRequest(r, "id")
	if err != nil {
		app.notFound(w)

		return
	}

	calls, err := app.callRepository.FindAllByCommand(command)
	if err != nil {
		app.serverError(w, err)

		return
	}

	err = app.inertiaManager.Render(w, r, "call/History", map[string]interface{}{
		"command": command,
		"calls":   calls,
	})
	if err != nil {
		app.serverError(w, err)
	}
}

func (app *App) callDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		app.methodNotAllowed(w, []string{"DELETE"})

		return
	}

	call, err := app.getCallFromRequest(r, "id")
	if err != nil {
		app.notFound(w)

		return
	}

	err = app.callRepository.Delete(call)
	if err != nil {
		app.serverError(w, err)

		return
	}

	app.sessionManager.Put(r.Context(), sessionKeyFlashMessage, "Deleted successfully.")
	http.Redirect(w, r, fmt.Sprintf("/call/history?id=%v", call.CommandId), http.StatusSeeOther)
}

func (app *App) getCallFromRequest(r *http.Request, parameter string) (*models.Call, error) {
	if r.URL.Query().Get(parameter) == "" {
		return nil, errors.New(fmt.Sprintf("%s parameter not found", parameter))
	}

	id, err := strconv.ParseInt(r.URL.Query().Get(parameter), 10, 64)
	if err != nil {
		return nil, err
	}

	call, err := app.callRepository.Find(id)
	if err != nil {
		return nil, err
	}

	return call, nil
}
