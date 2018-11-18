# Mongo optimistic FSM

Simple implementation of FSM to work with MongoDB.

## Notes

To prevent race conditions, optimistic locking is used. As with any technique, it has pros and cons:

Proc:
- no need to lock the whole document: reading and writing on some fields occurs without delay
- simple store interface (don't need of multi-statement transactions)

Cons:
- all or nothing: since there is no wait for the lock to be released (as well as the lock itself), status update operations may fail (with an error).

## Content

[store/mongo.go](mongo.go) file contains store engine implementation.

[main.go](main.go) file contains simple tests for concurrent state updates.
