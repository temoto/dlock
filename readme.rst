What
====

Dlock is a distributed lock manager [1]. It is designed after flock utility but for multiple machines. When client disconnects, all his locks are lost. TCP keep alive probes and optional protocol level heart beat ensure connection problems are detected in time.

dlock-server manages locks in memory, no persistence.
dlock client connects to server, sends a lock acquiring request, optionally waits if locks are being held by someone else.


How
===

Client-server speak very simple protocol built on Protocol Buffers [2] frames on top of TCP. Pipelining many requests before reading response is perfectly fine. Responses come in the order of requests.

Protocol::

    Length-prefixed protocol buffers. Length prefix is 4 bytes, big endian binary encoding.

    message Request {
        uint32 version = 1 [default = 2];
        uint64 id = 2;
        string access_token = 3;
        RequestType type = 4;

        // Ping is empty
        RequestLock lock = 51;
        // RequestUnlock unlock = 52;
    }

    message Response {
        uint32 version = 1 [default = 2];
        uint64 request_id = 2;
        ResponseStatus status = 3;
        string error_text = 4;
        repeated string keys = 5;
        int64 server_unix_time = 6; // Unix timestamp
    }

    As of 2013-05-28, API version is 2.

    Lock request:

    `type = Lock`

    message RequestLock {
        uint64 wait_micro = 1;
        uint64 release_micro = 2;
        repeated string keys = 3;
    }

Supplied keys are locked until client disconnects or for `release_micro` microseconds. If release timeout is supplied, disconnect does not do anything. If some of specified keys are already locked, this command will block for at most `wait_micro` microseconds before returning response with `AcquireTimeout` status.

Ping request::

    `type = Ping`

Response is always `Ok`. Clients should send pings periodically to inform server they are alive. Otherwise the server will disconnect them with suspection of failure and release their locks.

::

    enum RequestType {
        Ping = 1;
        Lock = 2;
    //  Unlock = 3;
    }

    enum ResponseStatus {
        // Error codes:
        // 1-99: protocol level errors
        // 100-119: [lock] input validation errors
        // 120-139: [lock] response errors for valid input
        Ok = 0;
        General = 1; // generic error, read message for details
        Version = 2; // incompatible request version
        InvalidType = 3; // unknown request type

        // Lock 100-199
        TooManyKeys = 100;
        AcquireTimeout = 120;
    }


References
==========

[1] http://en.wikipedia.org/wiki/Distributed_lock_manager

[2] https://code.google.com/p/protobuf/


Flair
=====

.. image:: https://travis-ci.org/temoto/dlock.svg?branch=master
    :target: https://travis-ci.org/temoto/dlock

.. image:: https://codecov.io/gh/temoto/dlock/branch/master/graph/badge.svg
    :target: https://codecov.io/gh/temoto/dlock
