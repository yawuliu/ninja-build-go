package main


func NewState()* State {
	ret :=State{}
	ret.bindings_.AddRule(&kPhonyRule);
	ret.AddPool(&kDefaultPool);
	ret.AddPool(&kConsolePool);
	return &ret
}

func (this* State) AddPool(pool *Pool) {
  if (this.LookupPool(pool.name()) == nil) {
      panic("AddPool")
  }
    this.pools_[pool.name()] = pool;
}

 func (this* State) LookupPool(pool_name string) *Pool {
  map<string, Pool*>::iterator i = pools_.find(pool_name);
  if (i == pools_.end()) {
      return nil
  }
  return i.second;
}

 func (this* State) AddEdge(rule *Rule)*Edge {
  edge := NewEdge();
  edge.rule_ = rule;
  edge.pool_ = &kDefaultPool;
  edge.env_ = &this.bindings_;
  edge.id_ = this.edges_.size();
  this.edges_.push_back(edge);
  return edge;
}

func (this* State) GetNode( path string, slash_bits uint64) *Node{
  node := this.LookupNode(path);
  if (node) {
	  return node
  }
  node = NewNode(path.AsString(), slash_bits);
  this.paths_[node.path()] = node;
  return node;
}

func (this* State) LookupNode( path string) *Node  {
  Paths::const_iterator i = this.paths_.find(path);
  if (i != this.paths_.end()) {
	  return i. second
  }
  return nmil
}

 func (this* State) SpellcheckNode(path string) * Node{
  kAllowReplacements := true;
  kMaxValidEditDistance := 3;

  min_distance := kMaxValidEditDistance + 1;
  var  result *Node = nil
  for (Paths::iterator i = paths_.begin(); i != paths_.end(); ++i) {
    distance := EditDistance(
        i.first, path, kAllowReplacements, kMaxValidEditDistance);
    if (distance < min_distance && i.second) {
      min_distance = distance;
      result = i.second;
    }
  }
  return result;
}

func (this* State) AddIn(edge *Edge,  path string, slash_bits  uint64) {
  node := this.GetNode(path, slash_bits);
  node.set_generated_by_dep_loader(false);
  edge.inputs_.push_back(node);
  node.AddOutEdge(edge);
}

func (this* State) AddOut( edge *Edge,  path string, slash_bits uint64, err *string) bool {
  node := this.GetNode(path, slash_bits);
  other := node.in_edge()
  if other!=nil {
    if (other == edge) {
      *err = path.AsString() + " is defined as an output multiple times";
    } else {
      *err = "multiple rules generate " + path.AsString();
    }
    return false;
  }
  edge.outputs_.push_back(node);
  node.set_in_edge(edge);
  node.set_generated_by_dep_loader(false);
  return true;
}

func (this* State) AddValidation(edge *Edge,  path string, slash_bits uint64) {
  node := GetNode(path, slash_bits);
  edge.validations_.push_back(node);
  node.AddValidationOutEdge(edge);
  node.set_generated_by_dep_loader(false);
}

func (this* State) AddDefault( path string,  err *string) bool {
  node := this.LookupNode(path);
  if (!node) {
    *err = "unknown target '" + path.AsString() + "'";
    return false;
  }
    this.defaults_.push_back(node);
  return true;
}

func (this* State) RootNodes(err *string) []*Node {
   root_nodes := []*Node{}
  // Search for nodes with no output.
  for (vector<Edge*>::const_iterator e = edges_.begin();
       e != edges_.end(); ++e) {
    for (vector<Node*>::const_iterator out = (*e).outputs_.begin();
         out != (*e).outputs_.end(); ++out) {
      if ((*out).out_edges().empty()) {
          root_nodes.push_back(*out)
      }
    }
  }

  if (!edges_.empty() && root_nodes.empty())
    *err = "could not determine root nodes of build graph";

  return root_nodes;
}

func (this* State) DefaultNodes( err *string) []*Node  {
  return this.defaults_.empty() ? this.RootNodes(err) : this.defaults_;
}

func (this* State) Reset() {
  for (Paths::iterator i = paths_.begin(); i != paths_.end(); ++i)
    i.second.ResetState();
  for (vector<Edge*>::iterator e = this.edges_.begin(); e != this.edges_.end(); ++e) {
    (*e).outputs_ready_ = false;
    (*e).deps_loaded_ = false;
    (*e).mark_ = Edge::VisitNone;
  }
}

func (this* State) Dump() {
  for (Paths::iterator i = paths_.begin(); i != paths_.end(); ++i) {
    node := i.second;
    printf("%s %s [id:%d]\n",
           node.path(),
           node.status_known() ? (node.dirty() ? "dirty" : "clean")
                                : "unknown",
           node.id());
  }
  if (!pools_.empty()) {
    printf("resource_pools:\n");
    for (map<string, Pool*>::const_iterator it = pools_.begin(); it != pools_.end(); it++) {
      if (!it.second.name().empty()) {
        it.second.Dump();
      }
    }
  }
}