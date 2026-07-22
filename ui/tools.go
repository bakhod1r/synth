package ui

// The hashing and encoding tools that used to live here have been removed.
//
// They were a separate utility bolted onto a data generator, and everything
// they did for generated data is now a column setting instead — mask=hash or
// mask=token, with salt=, secret= and digest= — applied at generation time
// rather than by hand afterwards.
//
// This file is kept empty rather than deleted so the removal is visible in
// review instead of silent.
