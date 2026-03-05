package model

import "errors"

var (
	ErrUserIDNotSpecified  = errors.New("empty user ID")
	ErrUserIDNotFound      = errors.New("user ID not found")
	ErrEventIDNotSpecified = errors.New("empty event ID")
	ErrNothingToDelete     = errors.New("empty user and event IDs for deletion")
	ErrEventNotFound       = errors.New("specified event ID for user ID not found")
	ErrDateNotSpecified    = errors.New("empty event date")
	ErrEventNotSpecified   = errors.New("empty event description")
	ErrNothingToUpdate     = errors.New("empty date and description for event update")
	ErrNothingToCreate     = errors.New("empty info to update event")
	ErrEventTimePast       = errors.New("event scheduled time is in the past")
	ErrEventDescrEmpty     = errors.New("event description is empty")
	ErrNoEventInQueue      = errors.New("currently no events are available")
	ErrCommon500           = errors.New("something went wrong. try again later")
)
