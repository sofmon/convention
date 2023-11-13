import '../lib/convention.dart' as conv;

class User {
  final String UserID;
  final String UserName;

  User(this.UserID, this.UserName);
}

class API extends conv.Client {
  late conv.Out<String> GetHealth;
  late conv.In<User> PutUser;
  late conv.Out<User> GetUser;
  late conv.InOut<User, User> PostUserDisable;

  API(String host, int port) : super(host, port) {
    GetHealth = conv.Out<String>(this, "GET /health", "");
    PutUser = conv.In<User>(this, "PUT /users/{user_id}", "");
    GetUser = conv.Out<User>(this, "GET /users/{user_id}", "");
    PostUserDisable = conv.InOut<User, User>(this, "POST /users/{user_id}/@block", "");
  }
}
