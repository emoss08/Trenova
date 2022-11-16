#!/usr/bin/env bash
: '
COPYRIGHT 2022 MONTA

This file is part of Monta.

Monta is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Monta is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Monta.  If not, see <https://www.gnu.org/licenses/>.

-------------------------------------------------------------------------------

This script is used to convert static image files to AVIF format.
'
echo 'Starting up...'
echo 'Starting redis...'
sudo service redis-server start
echo 'Starting postgres...'
sudo service postgresql start
echo 'Activating virtual environment...'
source rvenv/bin/activate.fish
echo 'Starting server'
py manage.py runserver
