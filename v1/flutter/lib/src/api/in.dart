import 'client.dart';

class _Descriptor {}

class In<inT> {
  final Client _client;
  final String _pattern;
  final String _description;

  const In(this._client, this._pattern, this._description);
}

class Out<outT> {
  final Client _client;
  final String _pattern;
  final String _description;

  const Out(this._client, this._pattern, this._description);
}

class InOut<inT, outT> {
  final Client _client;
  final String _pattern;
  final String _description;

  const InOut(this._client, this._pattern, this._description);
}

class Trigger {
  final Client _client;
  final String _pattern;
  final String _description;

  const Trigger(this._client, this._pattern, this._description);
}
