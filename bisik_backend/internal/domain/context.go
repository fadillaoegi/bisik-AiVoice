package domain

import "context"

// Context di-alias supaya layer domain tidak mengimpor paket luar
// secara langsung di signature repository.
type Context = context.Context
