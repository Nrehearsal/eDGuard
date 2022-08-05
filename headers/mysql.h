#ifndef MYSQL
#define MYSQL

#define COM_QUERY 3

#define DISPATCH_COMMAND_V57_FAILED -2

// mysql 5.7
// https://github.com/mysql/mysql-server/blob/5.7/include/mysql/com_data.h
struct COM_QUERY_DATA {
    const char *query;
    unsigned int length;
    //  struct PS_PARAM *parameters;    TODO
    //  unsigned long parameter_count;
};

struct st_mysql_const_lex_string
{
  const char *str;
  unsigned long length;
};

#define MIN_STR_LEN(a, b) (((a + 1) < (b)) ? (a + 1) : (b))

#define DEBUG_ENABLE

#endif
