What
====

Dlock is a distributed lock manager [1]. It is designed after flock utility but for multiple machines. When client disconnects, all his locks are lost. TCP keep alive probes and optional protocol level heart beat ensure connection problems are detected in time.

dlock-server manages locks in memory, no persistence.
dlock client connects to server, sends a lock acquiring request, optionally waits if locks are being held by someone else.


How
===

Client-server speak very simple protocol built on tnetstring [2] frames on top of TCP. Pipelining many requests before reading response is perfectly fine. Responses come in the order of requests.

Protocol:

	Lock request:

    [API version, 'lock', wait, release after, keys]

    API version - int. As of 2012-04-04, latest version is 1.
    wait - float, >= 0
    release after - false | float > 0
    key - string

    Supplied keys are locked until client disconnects or for `release after` seconds. If release timeout is supplied, disconnect does not do anything.

    Success response:
    ["ok"]

    Wait timeout response:
    ["error", 120, "Acquire timeout", list of keys not locked]

    Other error response:
    ["error", code, message[, data] ]

    code - int
    message - string


    Ping request:

    [API version, 'ping']

    Success response:
    ["ok"]


Error codes:

1-99: protocol level errors
100-119: [lock] input validation errors
120-139: [lock] response errors for valid input

More specifically:

1: generic error, read message for details
2: unsupported API version
3: unknown message type (command)
4: invalid number of arguments for given message type

Lock command:

100: invalid wait timeout
101: invalid release timeout
102: too many keys
103: some keys are not strings
120: acquire timeout


References
==========

[1] http://en.wikipedia.org/wiki/Distributed_lock_manager
[2] http://tnetstrings.org/