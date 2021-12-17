#!/bin/bash
#
# Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

function camelCase {
  sed -r 's/(.)-+(.)/\1\U\2/g;s/^[a-z]/\U&/' <<< "$1"
}

package_name="${1:-charts}"

out="
$(cat "$(dirname $0)/LICENSE_BOILERPLATE.txt" | sed "s/YEAR/$(date +%Y)/g")

// Code generated by $(basename $0). DO NOT EDIT.

package $package_name

const ("

for image_name in $(yaml2json < "images.yaml" | jq -r '[.images[].name] | unique | .[]'); do
  variable_name="$(camelCase "$image_name")"

  out="
$out
	// ImageName$variable_name is a constant for an image in the image vector with name '$image_name'.
	ImageName$variable_name = \"$image_name\""
done

out="
$out
)
"

echo "$out" > "images.go"
goimports -l -w "images.go"
