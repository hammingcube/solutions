// Copyright (c) 2015 Elements of Programming Interviews. All rights reserved.

#include <cassert>
#include <iostream>
#include <random>
#include <string>

#include "./Can_string_be_palindrome_sorting.h"

using std::cout;
using std::cin;
using std::endl;
using std::string;


int main(int argc, char *argv[]) {
  string s;
  while(cin >> s){
  	cout << CanStringBeAPalindromeSorting::CanStringBeAPalindrome(&s) << endl;
  }
  return 0;
}
