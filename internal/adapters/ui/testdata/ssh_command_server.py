# Copyright 2025.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Loopback-only SSH server: return the exec request without executing it."""

import socket
import threading

import paramiko


class CommandServer(paramiko.ServerInterface):
    def __init__(self):
        self.ready = threading.Event()
        self.command = None

    def check_auth_none(self, username):
        return paramiko.AUTH_SUCCESSFUL

    def check_channel_request(self, kind, channel_id):
        if kind == "session":
            return paramiko.OPEN_SUCCEEDED
        return paramiko.OPEN_FAILED_ADMINISTRATIVELY_PROHIBITED

    def check_channel_exec_request(self, channel, command):
        self.command = command
        self.ready.set()
        return True


key = paramiko.RSAKey.generate(2048)
with socket.socket() as listener:
    listener.bind(("127.0.0.1", 0))
    listener.listen(1)
    listener.settimeout(15)
    print(listener.getsockname()[1], flush=True)
    connection, _ = listener.accept()
    with connection, paramiko.Transport(connection) as transport:
        transport.add_server_key(key)
        server = CommandServer()
        transport.start_server(server=server)
        channel = transport.accept(10)
        if channel is None or not server.ready.wait(10):
            raise RuntimeError("SSH client did not send an exec request")
        channel.sendall(server.command)
        channel.send_exit_status(0)
        channel.close()
        # Let the client finish the SSH disconnect before closing TCP. Closing
        # while its channel-close packet is unread can reset the connection.
        transport.join(10)
