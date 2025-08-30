package query

import (
	"github.com/mudgallabs/tantra/apires"
	"github.com/mudgallabs/tantra/service"
)

type Cursor struct {
	After  *string `query:"after" schema:"after" json:"after"`
	Before *string `query:"before" schema:"before" json:"before"`
	Limit  *int    `query:"limit" schema:"limit" json:"limit,omitempty"`
}

func (cursor *Cursor) Validate(maxLimit, defaultLimit int) error {
	if cursor.Limit == nil || (cursor.Limit != nil && *cursor.Limit <= 0) {
		cursor.Limit = &defaultLimit
	}

	if *cursor.Limit > maxLimit {
		cursor.Limit = &maxLimit
	}

	var errs service.InputValidationErrors

	if cursor.AfterIsValid() && cursor.BeforeIsValid() {
		errs.Add(apires.NewApiError("Invalid cursor", "Cannot have both 'after' and 'before' set", "cursor", cursor))
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func (cursor *Cursor) AfterIsValid() bool {
	return cursor.After != nil && *cursor.After != ""
}

func (cursor *Cursor) BeforeIsValid() bool {
	return cursor.Before != nil && *cursor.Before != ""
}
