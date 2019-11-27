# Project TODO's
## Code cleaning and simplifications
* Handle some code duplication
  * forwardX functionality - currently has multiple such methods depending on the type of message being sent; consolidate into one common method
  * file downloading functionality - regular vs. chunked (e.g. result from a file search)
* Some long methods can be broken into separate smaller ones called from within
* Simplify some of the state handling and data structures - e.g. downloading state, file information storage, file searching
* Remove finished downloading states
* Deal with all the type mismatch/conversions - e.g. uint32, uint64, etc
## Functionality addition/changes
* Add usage of RLock in place of Lock where appropriate
* Add some error checks - e.g. indexing non-existing files
* Use some receiver methods where appropriate
