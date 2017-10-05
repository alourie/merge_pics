#!/opt/boxen/homebrew/bin/python
import re
import glob
import os
import sys
import shutil
import datetime
import exifread
import hashlib
import argparse


def fhash(filepath):
    BLOCKSIZE = 65536
    hasher = hashlib.sha256()
    with open(filepath, 'rb') as afile:
        buf = afile.read(BLOCKSIZE)
        while len(buf) > 0:
            hasher.update(buf)
            buf = afile.read(BLOCKSIZE)

    return hasher.hexdigest()

def check_containing(name):
    pattern = re.compile("IMG_(?P<date>[0-9]+)_(?P<time>[0-9]+)")
    match = pattern.search(name)
    if match:
        pdate = match.group('date')
        ptime = match.group('time')
        try:
            ndate = datetime.datetime.strptime(pdate, '%Y%m%d')
            ntime = datetime.datetime.strptime(ptime, '%H%M%S')
            return "{year}:{month}:{day} {hour}:{minute}:{sec}".format(
                year=ndate.year,
                month=ndate.month,
                day=ndate.day,
                hour=ntime.hour,
                minute=ntime.minute,
                sec=ntime.second
            )
        except ValueError:
            raise Exception("Error in date folder")
    else:
        return "1995:01:01 01:00:00"


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Params for the picsorter.')
    parser.add_argument('--target', help="Target folder")
    args = parser.parse_args()
    if args.target:
        targetdir = args.target
    else:
        targetdir = os.path.expanduser("~/Pictures/Australia")

    names = [y for x in os.walk(".") for y in glob.glob(os.path.join(x[0], '*.jpg'))]

    for name in names:
        with open(name, 'rb') as f:
            tags = exifread.process_file(f)
            if 'EXIF DateTimeOriginal' in tags:
                a = tags['EXIF DateTimeOriginal'].values
            elif 'Image DateTime' in tags:
                a = tags['Image DateTime'].values
            else:
                a = check_containing(name.split('/')[-2])

            date = datetime.datetime.strptime(a, '%Y:%m:%d %H:%M:%S')
            new_dir = os.path.join(targetdir, str(date.year), str(date.month) if date.month > 9 else "0" + str(date.month), str(date.day))

            if "IMG_" in name.split('/')[-2]:
                new_dir = os.path.join(new_dir, name.split('/')[-2])

            new_name = os.path.join(new_dir, name.split('/')[-1])
            picname, _ = name.split("/")[-1].split(".")

            if os.path.exists(new_name):
                # if yes, compare the hashes.
                if fhash(name) == fhash(new_name):
                    print "File {name} exists, removing".format(name=name)
                    os.remove(name)
                else:
                    new_name = os.path.join(new_dir, picname + "_01.jpg" + picname)
                    print "File {name} exists but different, copy to a new name".format(name=name)
                    shutil.copy(name, new_name)
            else:
                # if not, move the file
                if not os.path.exists(new_dir):
                    os.makedirs(new_dir)
                print "Copy {name} to the library".format(name=name)
                shutil.move(name, new_name)
