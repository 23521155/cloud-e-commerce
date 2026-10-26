self.__MIDDLEWARE_MATCHERS = [
  {
    "regexp": "^(?:\\/(_next\\/data\\/[^/]{1,}))?(?:\\/((?!_next\\/static|_next\\/image|images\\/|favicon.ico|icon.svg|logo.png|.*\\.(?:svg|png|jpg|jpeg|webp|ico)$).*))(\\.json|\\.rsc|\\.segments\\/.+\\.segment\\.rsc)?[\\/#\\?]?$",
    "originalSource": "/((?!_next/static|_next/image|images/|favicon.ico|icon.svg|logo.png|.*\\.(?:svg|png|jpg|jpeg|webp|ico)$).*)"
  }
];self.__MIDDLEWARE_MATCHERS_CB && self.__MIDDLEWARE_MATCHERS_CB()