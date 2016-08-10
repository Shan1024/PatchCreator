#!/usr/bin/env bash

patchDir="/home/shan/Desktop/patches-160630"
echo "Patch Directory: $patchDir"

distDir="/home/shan/Desktop/Test/wso2esb-4.9.0"
echo "Distribution Directory: $distDir"

patchNamePrefix="WSO2-CARBON-PATCH-4.4.0"

ARRAY=(0001 0002 0003 0004 0005 0006 0009 0012 0014 0020 0022 0028 0029 0034 0038 0039 0043 0046 0049 0052 0063 0095 0097 0101 0111 0112 0114 0115 0118 0120 0123 0133 0144 0151 0152 0164 0166 0167 0168 0169 0171 0174 0177 0184 0187 0193 0194 0195 0197 0200 0202 0205 0206 0217 0218 0219 0237 0251 0254 0258 0262 0270 0272 0284 0286 0288 0292 0302 0309 0313 0315 0318 0319 0346 0353 0369)
# get number of elements in the array
ELEMENTS=${#ARRAY[@]}

STRING=""
for ((i=0;i<$ELEMENTS;i++)); do
	echo
	echo "#########################################################################"
    echo "######################### Converting Patch ${ARRAY[${i}]} #########################"
    echo "#########################################################################"
    uct create $patchDir/$patchNamePrefix-${ARRAY[${i}]}/patch${ARRAY[${i}]} $distDir
    echo "########################################################################"
    echo
done
