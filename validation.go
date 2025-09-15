package rq

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Validator is a function that validates a response
type Validator func(*Response) error

// Validate adds one or more validators to the request
func (r *Request) Validate(validators ...Validator) *Request {
	if r.err != nil {
		return r
	}
	r.validators = append(r.validators, validators...)
	return r
}

// Validate provides a namespace for validation functions
var Validate = validateNamespace{}

type validateNamespace struct{}

// OK validates that the response has a 2xx status code
func (validateNamespace) OK() Validator {
	return func(r *Response) error {
		if r.err != nil {
			return r.err
		}
		if !r.IsOK() {
			return fmt.Errorf("expected 2xx status, got %d", r.StatusCode)
		}
		return nil
	}
}

// Status validates that the response has the expected status code
func (validateNamespace) StatusCode(expected int) Validator {
	return func(r *Response) error {
		if r.err != nil {
			return r.err
		}
		if r.StatusCode != expected {
			return fmt.Errorf("expected status %d, got %d", expected, r.StatusCode)
		}
		return nil
	}
}

// Header validates that the response has a specific header with expected value
func (validateNamespace) Header(key, expectedValue string) Validator {
	return func(r *Response) error {
		if r.err != nil {
			return r.err
		}

		actualValue := r.Header.Get(key)
		if actualValue != expectedValue {
			return fmt.Errorf("expected header %q to be %q, got %q", key, expectedValue, actualValue)
		}

		return nil
	}
}

// HeaderExists validates that the response has a specific header (any value)
func (validateNamespace) HeaderExists(key string) Validator {
	return func(r *Response) error {
		if r.err != nil {
			return r.err
		}
		if r.Header.Get(key) == "" {
			return fmt.Errorf("expected header %q to exist", key)
		}
		return nil
	}
}

// BodyContains validates that the response body contains a specific substring
func (validateNamespace) BodyContains(substr string) Validator {
	return func(r *Response) error {
		if r.err != nil {
			return r.err
		}

		bodyStr := string(r.body)
		if !strings.Contains(bodyStr, substr) {
			return fmt.Errorf("response body does not contain %q", substr)
		}

		return nil
	}
}

// BodyMatches validates that the response body matches a regex pattern
func (validateNamespace) BodyMatches(pattern string) Validator {
	return func(r *Response) error {
		if r.err != nil {
			return r.err
		}

		matched, err := regexp.Match(pattern, r.body)
		if err != nil {
			return fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
		}
		if !matched {
			return fmt.Errorf("response body does not match pattern %q", pattern)
		}

		return nil
	}
}

// All combines multiple validators - all must pass
func (validateNamespace) All(validators ...Validator) Validator {
	return func(r *Response) error {
		for _, validator := range validators {
			if err := validator(r); err != nil {
				return err
			}
		}
		return nil
	}
}

// Any returns success if any of the validators pass
func (validateNamespace) Any(validators ...Validator) Validator {
	return func(r *Response) error {
		var errs []error

		for _, validator := range validators {
			if err := validator(r); err == nil {
				return nil // At least one validator passed
			} else {
				errs = append(errs, err)
			}
		}

		// All validators failed
		if len(errs) == 1 {
			return errs[0]
		}

		var errStr strings.Builder
		errStr.WriteString("all validators failed:")
		for i, err := range errs {
			errStr.WriteString(fmt.Sprintf(" [%d] %v", i+1, err))
		}

		return errors.New(errStr.String())
	}
}

// Not inverts a validator (success becomes failure and vice versa)
func (validateNamespace) Not(validator Validator) Validator {
	return func(r *Response) error {
		if err := validator(r); err == nil {
			return fmt.Errorf("expected validation to fail but it passed")
		}
		return nil
	}
}
