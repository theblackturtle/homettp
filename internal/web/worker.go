package web

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/homettp/homettp/internal/models"
)

func (app *App) worker() {
	for id := range app.queue {
		err := app.handleCall(id)
		if err != nil {
			app.errorLog.Print(err)
		}
	}
}

func (app *App) handleCall(id int64) error {
	call, err := app.callRepository.Find(id)
	if err != nil {
		return err
	}

	if call.Status != models.Pending {
		return errors.New(fmt.Sprintf("worker: invalid call model %v", id))
	}

	err = app.callRepository.Update(call, &models.Call{
		Status: models.InProgress,
	})
	if err != nil {
		return err
	}

	command, err := app.commandRepository.Find(call.CommandId)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(app.commandTimeout)*time.Second)
	defer cancel()

	segments := strings.Fields(strings.ReplaceAll(command.Value, models.PayloadVariable, call.Payload))
	name, arg := segments[0], segments[1:]

	cmd := exec.CommandContext(ctx, name, arg...)
	out, err := cmd.Output()

	if ctx.Err() == context.DeadlineExceeded {
		err = app.callRepository.Update(call, &models.Call{
			Status: models.Failed,
			Output: "command timed out",
		})
		if err != nil {
			return err
		}
	} else if err != nil {
		err = app.callRepository.Update(call, &models.Call{
			Status: models.Failed,
			Output: err.Error(),
		})
		if err != nil {
			return err
		}
	} else {
		err = app.callRepository.Update(call, &models.Call{
			Status: models.Completed,
			Output: string(out),
		})
		if err != nil {
			return err
		}
	}

	return nil
}
