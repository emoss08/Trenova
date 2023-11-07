# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  The Monta software is licensed under the Business Source License 1.1. You are granted the right -
#  to copy, modify, and redistribute the software, but only for non-production use or with a total -
#  of less than three server instances. Starting from the Change Date (November 16, 2026), the     -
#  software will be made available under version 2 or later of the GNU General Public License.     -
#  If you use the software in violation of this license, your rights under the license will be     -
#  terminated automatically. The software is provided "as is," and the Licensor disclaims all      -
#  warranties and conditions. If you use this license's text or the "Business Source License" name -
#  and trademark, you must comply with the Licensor's covenants, which include specifying the      -
#  Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use     -
#  Grant, and not modifying the license in any other way.                                          -
# --------------------------------------------------------------------------------------------------
import logging
import os
import shutil

from pyftpdlib.authorizers import DummyAuthorizer
from pyftpdlib.handlers import FTPHandler
from pyftpdlib.servers import FTPServer


class MontaFTPHandler(FTPHandler):
    def on_file_received(self, file: str) -> None:
        logging.info(f"File received: {file}")
        try:
            processing_folder = os.path.abspath(
                os.path.join(os.path.dirname(file), "../process")
            )
            shutil.move(file, f"{processing_folder}/{os.path.basename(file)}")
        except Exception as e:
            logging.error(f"An error occurred while moving the file: {e}")

    def on_incomplete_file_received(self, file: str) -> None:
        try:
            os.remove(file)
        except Exception as e:
            logging.error(f"An error occurred while deleting the incomplete file: {e}")


def run_server() -> None:
    authorizer = DummyAuthorizer()
    authorizer.add_user("user", "12345", ".", perm="elradfmwMT")
    handler = MontaFTPHandler
    handler.authorizer = authorizer
    handler.banner = "Monta FTP Server"

    logging.basicConfig(filename="ftp.log", level=logging.INFO)

    server = FTPServer(("127.0.0.1", 2121), handler=handler)
    server.serve_forever()


if __name__ == "__main__":
    run_server()
