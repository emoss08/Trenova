class LocalStorageService {
  // static property to store the key for the token in local storage
  public static TOKEN = 'token';
  // static property to store the key for the user info in local storage
  public static USER = 'm_user_info';

  /**
   * Method to store the given token in local storage
   * @param {string} token - the token to store
   * @returns {void}
   */
  public static setToken(token: string) {
    localStorage.setItem(LocalStorageService.TOKEN, token);
  }

  /**
   * Method to retrieve the token from local storage
   * @returns {string} the token stored in local storage
   */
  public static getToken() {
    return localStorage.getItem(LocalStorageService.TOKEN);
  }

  /**
   * Method to store the given user info in local storage
   * @param {any} user - the user info to store
   */
  public static setUser(user: any) {
    localStorage.setItem(LocalStorageService.USER, JSON.stringify(user));
  }

  /**
   * Method to retrieve the user info from local storage
   * @returns {any} the user info stored in local storage
   */
  public static getUser() {
    const user = localStorage.getItem(LocalStorageService.USER);
    return user ? JSON.parse(user) : null;
  }

  /**
   * Method to remove the token from local storage
   */
  public static removeToken() {
    localStorage.removeItem(LocalStorageService.TOKEN);
  }

  /**
   * @description Method to remove the user info from local storage
   */
  public static removeUser() {
    localStorage.removeItem(LocalStorageService.USER);
  }

  /**
   * @description Method to remove both the token and user info from local storage
   */
  public static clearRelatedUser() {
    localStorage.removeItem(LocalStorageService.TOKEN);
    localStorage.removeItem(LocalStorageService.USER);
  }

  /**
   * Method to check if there is a token stored in local storage
   * @returns {boolean} indicating whether there is a token stored in local storage
   */
  public static hasToken() {
    return !!LocalStorageService.getToken();
  }

  /**
   * Method to check if there is user info stored in local storage
   * @returns {boolean} indicating whether there is user info stored in local storage
   */
  public static hasUser() {
    return !!LocalStorageService.getUser();
  }
}

export default LocalStorageService;
