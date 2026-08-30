export interface User {
  id: number;
  username: string;
  createdAt: string;
  updatedAt: string;
}

export interface Credentials {
  username: string;
  password: string;
}

export interface TokenPair {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  user: User;
}