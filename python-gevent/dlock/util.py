# coding: utf-8
from datetime import datetime
import errno
import gevent
import sys
import tnetstring


_missing = object()


class IdleTimeout(gevent.Timeout):
    pass


class ReadTimeout(gevent.Timeout):
    pass


def log(level, msg, **kwargs):
    s = u"{now} {level} {message}\n".format(now=datetime.now().isoformat(), level=level, message=msg.format(**kwargs))
    sys.stderr.write(s.encode('utf-8', 'replace'))
    sys.stderr.flush()


def ignore_epipe(f, *args, **kwargs):
    try:
        f(*args, **kwargs)
    except IOError as e:
        if e.errno != errno.EPIPE:
            raise


def read_until(f, delim, size, idle_timeout, read_timeout):
    """Repeat `f.read(1)` until EOF or total length > size or delim encountered.
    `idle_timeout` limits waiting for first byte of message.
    `read_timeout` limits waiting for the rest.
    On timeouts appropriate (Idle|Read)Timeout exception is raised.
    On EOF returns empty string.
    """
    buf = bytearray(size)
    length = 0

    with IdleTimeout(idle_timeout):
        s = f.read(1)

    if s:
        buf[length] = s
        length += 1
    else:
        return s

    with ReadTimeout(read_timeout):
        while length < size:
            s = f.read(1)
            if s:
                if s == delim:
                    break
                buf[length] = s
                length += 1
            else:
                break

    del buf[length:]
    return buf


def read_message(f, idle_timeout, read_timeout, max_length):
    """File-like object -> command, error
    If command is None, error happened.
    If error is not None, encode it and send to client.
    """
    length_str = read_until(f, ":", 10, idle_timeout, read_timeout)

    # EOF
    if length_str == "":
        return None, None

    try:
        length = int(length_str)
    except ValueError:
        return None, "Invalid length prefix"

    if length > max_length:
        return None, "Too long tnetstring"

    with ReadTimeout(read_timeout):
        rest = f.read(length + 1)

    # May raise many different ValueError-s.
    data = tnetstring.loads(str(length_str) + ":" + rest)

    if not isinstance(data, (tuple, list)):
        return None, "Invalid command data, expected list encoded in tnetstring"

    if not data:
        return None, "Empty command sequence"

    return data, None


def send_message(socket, data, timeout):
    s = tnetstring.dumps(data)

    with gevent.Timeout(timeout, False):
        try:
            socket.sendall(s)
        except IOError as e:
            if e.errno == errno.EPIPE:
                return False
            raise
        return True
    return False
