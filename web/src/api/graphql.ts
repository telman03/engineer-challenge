const GRAPHQL_URL = import.meta.env.VITE_GRAPHQL_URL || '/graphql';

interface GraphQLResponse<T> {
  data?: T;
  errors?: Array<{ message: string }>;
}

async function graphql<T>(query: string, variables?: Record<string, unknown>): Promise<T> {
  const token = localStorage.getItem('access_token');
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const res = await fetch(GRAPHQL_URL, {
    method: 'POST',
    headers,
    body: JSON.stringify({ query, variables }),
  });

  const json: GraphQLResponse<T> = await res.json();
  if (json.errors?.length) {
    throw new Error(json.errors[0].message);
  }
  if (!json.data) {
    throw new Error('No data returned');
  }
  return json.data;
}

export interface AuthResponse {
  accessToken: string;
  refreshToken: string;
  user: { id: string; email: string };
}

export interface RegisterResponse {
  id: string;
  email: string;
  status: string;
}

export async function register(email: string, password: string): Promise<AuthResponse> {
  // Register returns User, then we login to get tokens
  await graphql<{ register: RegisterResponse }>(`
    mutation Register($email: String!, $password: String!) {
      register(email: $email, password: $password) {
        id
        email
        status
      }
    }
  `, { email, password });
  // Auto-login after registration
  return login(email, password);
}

export async function login(email: string, password: string): Promise<AuthResponse> {
  const data = await graphql<{ login: AuthResponse }>(`
    mutation Login($email: String!, $password: String!) {
      login(email: $email, password: $password) {
        accessToken
        refreshToken
        user {
          id
          email
        }
      }
    }
  `, { email, password });
  return data.login;
}

export async function requestPasswordReset(email: string): Promise<{ success: boolean; token: string | null }> {
  const data = await graphql<{ requestPasswordReset: { success: boolean; token: string | null } }>(`
    mutation RequestPasswordReset($email: String!) {
      requestPasswordReset(email: $email) {
        success
        token
      }
    }
  `, { email });
  return data.requestPasswordReset;
}

export async function resetPassword(email: string, token: string, newPassword: string): Promise<boolean> {
  const data = await graphql<{ resetPassword: boolean }>(`
    mutation ResetPassword($email: String!, $token: String!, $newPassword: String!) {
      resetPassword(email: $email, token: $token, newPassword: $newPassword)
    }
  `, { email, token, newPassword });
  return data.resetPassword;
}

export async function refreshTokenMutation(token: string): Promise<{ accessToken: string; refreshToken: string }> {
  const data = await graphql<{ refreshToken: { accessToken: string; refreshToken: string } }>(`
    mutation RefreshToken($refreshToken: String!) {
      refreshToken(refreshToken: $refreshToken) {
        accessToken
        refreshToken
      }
    }
  `, { refreshToken: token });
  return data.refreshToken;
}
