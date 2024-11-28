package main


func NewState()* State {
	ret :=State{}
	ret.bindings_.AddRule(&kPhonyRule);
	ret.AddPool(&kDefaultPool);
	ret.AddPool(&kConsolePool);
	return &ret
}

func (this* State) AddPool(pool *Pool) {
  assert(LookupPool(pool->name()) == NULL);
  pools_[pool->name()] = pool;
}

 func (this* State) LookupPool(pool_name string) *Pool {
  map<string, Pool*>::iterator i = pools_.find(pool_name);
  if (i == pools_.end())
    return NULL;
  return i->second;
}

 func (this* State) AddEdge(rule *Rule)*Edge {
  Edge* edge = new Edge();
  edge->rule_ = rule;
  edge->pool_ = &State::kDefaultPool;
  edge->env_ = &bindings_;
  edge->id_ = edges_.size();
  edges_.push_back(edge);
  return edge;
}

func (this* State) GetNode( path StringPiece, slash_bits uint64) *Node{
  Node* node = LookupNode(path);
  if (node) {
	  return node
  }
  node = new Node(path.AsString(), slash_bits);
  paths_[node->path()] = node;
  return node;
}

func (this* State) LookupNode(StringPiece path) *Node  {
  Paths::const_iterator i = paths_.find(path);
  if (i != paths_.end()) {
	  return i- > second
  }
  return NULL;
}

 func (this* State) SpellcheckNode(path string) * Node{
  const bool kAllowReplacements = true;
  const int kMaxValidEditDistance = 3;

  int min_distance = kMaxValidEditDistance + 1;
  Node* result = NULL;
  for (Paths::iterator i = paths_.begin(); i != paths_.end(); ++i) {
    int distance = EditDistance(
        i->first, path, kAllowReplacements, kMaxValidEditDistance);
    if (distance < min_distance && i->second) {
      min_distance = distance;
      result = i->second;
    }
  }
  return result;
}

func (this* State) AddIn(edge *Edge,  path StringPiece, slash_bits  uint64) {
  Node* node = GetNode(path, slash_bits);
  node->set_generated_by_dep_loader(false);
  edge->inputs_.push_back(node);
  node->AddOutEdge(edge);
}

func (this* State) AddOut( edge *Edge,  path StringPiece, slash_bits uint64, err *string) bool {
  Node* node = GetNode(path, slash_bits);
  if (Edge* other = node->in_edge()) {
    if (other == edge) {
      *err = path.AsString() + " is defined as an output multiple times";
    } else {
      *err = "multiple rules generate " + path.AsString();
    }
    return false;
  }
  edge->outputs_.push_back(node);
  node->set_in_edge(edge);
  node->set_generated_by_dep_loader(false);
  return true;
}

func (this* State) AddValidation(edge *Edge,  path StringPiece, slash_bits uint64) {
  Node* node = GetNode(path, slash_bits);
  edge->validations_.push_back(node);
  node->AddValidationOutEdge(edge);
  node->set_generated_by_dep_loader(false);
}

func (this* State) AddDefault( path StringPiece,  err *string) bool {
  Node* node = LookupNode(path);
  if (!node) {
    *err = "unknown target '" + path.AsString() + "'";
    return false;
  }
  defaults_.push_back(node);
  return true;
}

func (this* State) RootNodes(err *string) []*Node {
  vector<Node*> root_nodes;
  // Search for nodes with no output.
  for (vector<Edge*>::const_iterator e = edges_.begin();
       e != edges_.end(); ++e) {
    for (vector<Node*>::const_iterator out = (*e)->outputs_.begin();
         out != (*e)->outputs_.end(); ++out) {
      if ((*out)->out_edges().empty())
        root_nodes.push_back(*out);
    }
  }

  if (!edges_.empty() && root_nodes.empty())
    *err = "could not determine root nodes of build graph";

  return root_nodes;
}

func (this* State) DefaultNodes( err *string) []*Node  {
  return defaults_.empty() ? RootNodes(err) : defaults_;
}

func (this* State) Reset() {
  for (Paths::iterator i = paths_.begin(); i != paths_.end(); ++i)
    i->second->ResetState();
  for (vector<Edge*>::iterator e = edges_.begin(); e != edges_.end(); ++e) {
    (*e)->outputs_ready_ = false;
    (*e)->deps_loaded_ = false;
    (*e)->mark_ = Edge::VisitNone;
  }
}

func (this* State) Dump() {
  for (Paths::iterator i = paths_.begin(); i != paths_.end(); ++i) {
    Node* node = i->second;
    printf("%s %s [id:%d]\n",
           node->path().c_str(),
           node->status_known() ? (node->dirty() ? "dirty" : "clean")
                                : "unknown",
           node->id());
  }
  if (!pools_.empty()) {
    printf("resource_pools:\n");
    for (map<string, Pool*>::const_iterator it = pools_.begin(); it != pools_.end(); it++) {
      if (!it->second->name().empty()) {
        it->second->Dump();
      }
    }
  }
}