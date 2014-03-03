/*
 * Copyright (c) 2005-2008, Message Systems, Inc.
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are
 * met:
 *
 *    * Redistributions of source code must retain the above copyright
 *      notice, this list of conditions and the following disclaimer.
 *    * Redistributions in binary form must reproduce the above
 *      copyright notice, this list of conditions and the following
 *      disclaimer in the documentation and/or other materials provided
 *      with the distribution.
 *    * Neither the name Message Systems, Inc. nor the names
 *      of its contributors may be used to endorse or promote products
 *      derived from this software without specific prior written
 *      permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
 * "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
 * LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
 * A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
 * OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
 * SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
 * LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
 * DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
 * THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
 * OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

#include <errno.h>
#include <dirent.h>
#include <stdint.h>
#include <stdio.h>
#include <unistd.h>

#include "jlog_config.h"
#include "jlog_private.h"
#include "getopt_long.h"

static int verbose = 0;
static int show_progress = 0;
static int show_subscribers = 0;
static int show_files = 0;
static int show_index_info = 0;
static int analyze_datafiles = 0;
static int repair_datafiles = 0;
static int cleanup = 0;
static int quiet = 0;
static char *add_subscriber = NULL;
static char *remove_subscriber = NULL;

#define OUT(...) if (!quiet) printf(__VA_ARGS__)

static void
usage(const char *prog)
{
  printf("Usage:\n    %s <options> logpath1 [logpath2 [...]]\n",
         prog);
  printf("\t-a <sub>:\tAdd <sub> as a log subscriber\n");
  printf("\t-e <sub>:\tErase <sub> as a log subscriber\n");
  printf("\t-p <sub>:\tShow the perspective of the subscriber <sub>\n");
  printf("\t      -l:\tList all log segments with sizes and readers\n");
  printf("\t      -i:\tList index information\n");
  printf("\t      -c:\tClean all log segments with no pending readers\n");
  printf("\t      -s:\tShow all subscribers\n");
  printf("\t      -d:\tAnalyze datafiles\n");
  printf("\t      -r:\tAnalyze datafiles and repair if needed\n");
  printf("\t      -v:\tVerbose output\n");
  printf("\nWARNING: the -r option can't be used on jlogs that are "
         "open by another process\n");
}

static int
is_datafile(const char *f, uint32_t *logid)
{
  uint32_t l = 0;
  int i;

  for (i = 0; i < 8; i++) {
    if ((f[i] >= '0' && f[i] <= '9') || (f[i] >= 'a' && f[i] <= 'f')) {
      l <<= 4;
      l |= (f[i] < 'a') ? (f[i] - '0') : (f[i] - 'a' + 10);
    } else {
      return 0;
    }
  }

  if (f[i] != '\0') {
    return 0;
  }

  if (logid) {
    *logid = l;
  }

  return 1;
}

static void
analyze_datafile(jlog_ctx *ctx, uint32_t logid)
{
  char idxfile[MAXPATHLEN];

  if (jlog_inspect_datafile(ctx, logid) > 0) {
    fprintf(stderr, "One or more errors were found.\n");

    if (repair_datafiles) {
      jlog_repair_datafile(ctx, logid);
      fprintf(stderr, "Log file reconstructed, deleting the corresponding idx file.\n");
      STRSETDATAFILE(ctx, idxfile, logid);
      strcat(idxfile, INDEX_EXT);
      unlink(idxfile);
    }
  }
}

static int
process_jlog(const char *file, const char *sub)
{
  jlog_ctx *log = jlog_new(file);

  if (add_subscriber) {
    if (jlog_ctx_add_subscriber(log, add_subscriber, JLOG_BEGIN)) {
      fprintf(stderr, "Could not add subscriber '%s': %s\n", add_subscriber,
              jlog_ctx_err_string(log));
    } else {
      OUT("Added subscriber '%s'\n", add_subscriber);
    }
  }

  if (remove_subscriber) {
    if (jlog_ctx_remove_subscriber(log, remove_subscriber) <= 0) {
      fprintf(stderr, "Could not erase subscriber '%s': %s\n",
              remove_subscriber, jlog_ctx_err_string(log));
    } else {
      OUT("Erased subscriber '%s'\n", remove_subscriber);
    }
  }

  if (!sub) {
    if (jlog_ctx_open_writer(log)) {
      fprintf(stderr, "error opening '%s'\n", file);
      return 0;
    }
  } else {
    if (jlog_ctx_open_reader(log, sub)) {
      fprintf(stderr, "error opening '%s'\n", file);
      return 0;
    }
  }

  if (show_progress) {
    char buff[20], buff2[20], buff3[20];
    jlog_id id, id2, id3;

    jlog_get_checkpoint(log, sub, &id);

    if (jlog_ctx_last_log_id(log, &id3)) {
      fprintf(stderr, "jlog_error: %s\n", jlog_ctx_err_string(log));
      fprintf(stderr, "error calling jlog_ctx_last_log_id\n");
    }

    jlog_snprint_logid(buff, sizeof(buff), &id);
    jlog_snprint_logid(buff3, sizeof(buff3), &id3);
    OUT("--------------------\n"
        "  Perspective of the '%s' subscriber\n"
        "    current checkpoint: %s\n"
        "    Last write: %s\n", sub, buff, buff3);

    if (jlog_ctx_read_interval(log, &id, &id2) < 0) {
      fprintf(stderr, "jlog_error: %s\n", jlog_ctx_err_string(log));
    }

    jlog_snprint_logid(buff, sizeof(buff), &id);
    jlog_snprint_logid(buff2, sizeof(buff2), &id2);
    OUT("    next interval: [%s, %s]\n"
        "--------------------\n\n", buff, buff2);
  }

  if (show_subscribers) {
    char **list;
    int i;

    jlog_ctx_list_subscribers(log, &list);

    for (i = 0; list[i]; i++) {
      char buff[20];
      jlog_id id;

      jlog_get_checkpoint(log, list[i], &id);
      jlog_snprint_logid(buff, sizeof(buff), &id);
      OUT("\t%32s @ %s\n", list[i], buff);
    }

    jlog_ctx_list_subscribers_dispose(log, list);
  }

  if (show_files) {
    struct dirent *de;
    DIR *dir;

    dir = opendir(file);

    if (!dir) {
      fprintf(stderr, "error opening '%s'\n", file);
      return 0;
    }

    while ((de = readdir(dir)) != NULL) {
      uint32_t logid;

      if (is_datafile(de->d_name, &logid)) {
        char fullfile[MAXPATHLEN];
        char fullidx[MAXPATHLEN];
        struct stat sb;
        int readers;

        snprintf(fullfile, sizeof(fullfile), "%s/%s", file, de->d_name);
        snprintf(fullidx, sizeof(fullidx), "%s/%s" INDEX_EXT, file, de->d_name);

        if (stat(fullfile, &sb)) {
          OUT("\t%8s [error stat(2)ing file: %s\n", de->d_name, strerror(errno));
        } else {
          readers = __jlog_pending_readers(log, logid);

          OUT("\t%8s [%ju bytes] %d pending readers\n", de->d_name, sb.st_size, readers);

          if (show_index_info) {
            if (stat(fullidx, &sb)) {
              OUT("\t\t idx: none\n");
            } else {
              uint32_t marker;
              int closed;

              if (jlog_idx_details(log, logid, &marker, &closed)) {
                OUT("\t\t idx: error\n");
              } else {
                OUT("\t\t idx: %u messages (%08x), %s\n", marker, marker, closed ? "closed" : "open");
              }
            }
          }
          
          if (analyze_datafiles) {
            analyze_datafile(log, logid);
          }

          if (readers == 0 && cleanup) {
            unlink(fullfile);
            unlink(fullidx);
          }
        }
      }
    }

    closedir(dir);
  }

  jlog_ctx_close(log);
}

int
main(int argc, char **argv)
{
  char *subscriber = NULL;
  int i, c;

  while ((c = getopt_long(argc, argv, "a:e:dsilrcp:v", NULL, NULL)) != EOF) {
    switch(c) {
    case 'v':
      verbose = 1;
      break;

    case 'i':
      show_files = 1;
      show_index_info = 1;
      break;

    case 'r':
      show_files = 1;
      analyze_datafiles = 1;
      repair_datafiles = 1;
      break;

    case 'd':
      show_files = 1;
      analyze_datafiles = 1;
      break;

    case 'a':
      add_subscriber = optarg;
      break;

    case 'e':
      remove_subscriber = optarg;
      break;

    case 'p':
      show_progress = 1;
      subscriber = optarg;
      break;

    case 's':
      show_subscribers = 1;
      break;

    case 'c':
      show_files = 1;
      quiet = 1;
      cleanup = 1;
      break;

    case 'l':
      show_files = 1;
      break;

    default:
      usage(argv[0]);
      exit(-1);
    }
  }

  if (optind == argc) {
    usage(argv[0]);
    exit(-1);
  }

  for (i = optind; i < argc; i++) {
    OUT("%s\n", argv[i]);
    process_jlog(argv[i], subscriber);
  }

  return 0;
}
/* vim: sts=2:ts=2:sw=2:et
 * */
