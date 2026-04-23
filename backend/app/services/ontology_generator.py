"""
"""

import json
import logging
import re
from typing import Dict, Any, List, Optional
from ..utils.llm_client import LLMClient
from ..utils.locale import get_language_instruction

logger = logging.getLogger(__name__)


def _normalize_attr_item(raw: Any) -> Optional[Dict[str, str]]:
    """LLMs sometimes return attribute names as plain strings; graph build expects dicts with 'name'."""
    if isinstance(raw, dict) and raw.get("name"):
        return raw
    if isinstance(raw, str) and raw.strip():
        s = raw.strip()
        # Safe snake_case-ish name (reserved names are fixed later in graph_builder)
        name = re.sub(r"[^a-zA-Z0-9_]+", "_", s).strip("_").lower() or "attribute"
        return {"name": name, "type": "text", "description": s}
    return None


def _normalize_source_target(raw: Any) -> Optional[Dict[str, str]]:
    if isinstance(raw, dict) and (raw.get("source") is not None or raw.get("target") is not None):
        return {
            "source": str(raw.get("source", "Entity") or "Entity"),
            "target": str(raw.get("target", "Entity") or "Entity"),
        }
    if isinstance(raw, (list, tuple)) and len(raw) >= 2:
        return {"source": str(raw[0]), "target": str(raw[1])}
    return None


def _to_pascal_case(name: str) -> str:
    parts = re.split(r'[^a-zA-Z0-9]+', name)
    words = []
    for part in parts:
        words.extend(re.sub(r'([a-z])([A-Z])', r'\1_\2', part).split('_'))
    result = ''.join(word.capitalize() for word in words if word)
    return result if result else 'Unknown'


ONTOLOGY_SYSTEM_PROMPT = """You are an expert ontology designer for knowledge graphs. Analyze the provided text and simulation requirement, then design entity and relationship types suitable for a social-media opinion simulation.

System framing:
- each entity should represent a real-world actor or account that can speak, interact, and spread information online
- entities should influence, repost, comment on, and respond to one another
- the ontology should support simulation of reactions, discourse, and information flow

Use real actors such as people, organizations, companies, media outlets, regulators, and communities. Avoid abstract concepts like sentiment, opinion, trends, or generic stances as entity types.

Return JSON with the fields `entity_types`, `edge_types`, and `analysis_summary`.

Entity rules:
- output exactly 10 entity types
- the last 2 must be fallback types `Person` and `Organization`
- the first 8 must be concrete text-derived types
- entity descriptions must be short English descriptions

Relationship rules:
- output 6-10 relationship types
- relationship names must be English `UPPER_SNAKE_CASE`
- each relationship must define valid `source_targets`

Attribute rules:
- use 1-3 key attributes per entity type
- each attribute must be an object with "name" and "description" keys, not a bare string in the array
- do not use reserved names such as `name`, `uuid`, `group_id`, `created_at`, or `summary`
- prefer names like `full_name`, `org_name`, `title`, `role`, `position`, `location`, and `description`
"""


class OntologyGenerator:
    """
    Ontology generator.
    Analyzes source text and creates entity and relationship type definitions.
    """
    
    def __init__(self, llm_client: Optional[LLMClient] = None):
        self.llm_client = llm_client or LLMClient()
    
    def generate(
        self,
        document_texts: List[str],
        simulation_requirement: str,
        additional_context: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Generate an ontology definition.

        Args:
            document_texts: list of source document texts
            simulation_requirement: simulation requirement description
            additional_context: additional optional context

        Returns:
            ontology definition including entity_types and edge_types
        """
        user_message = self._build_user_message(
            document_texts, 
            simulation_requirement,
            additional_context
        )
        
        lang_instruction = get_language_instruction()
        system_prompt = f"{ONTOLOGY_SYSTEM_PROMPT}\n\n{lang_instruction}\nIMPORTANT: Entity type names MUST be in English PascalCase (e.g., 'PersonEntity', 'MediaOrganization'). Relationship type names MUST be in English UPPER_SNAKE_CASE (e.g., 'WORKS_FOR'). Attribute names MUST be in English snake_case. Only description fields and analysis_summary should use the specified language above."
        messages = [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": user_message}
        ]
        
        result = self.llm_client.chat_json(
            messages=messages,
            temperature=0.3,
            max_tokens=4096
        )
        
        result = self._validate_and_process(result)
        
        return result
    
    MAX_TEXT_LENGTH_FOR_LLM = 50000
    
    def _build_user_message(
        self,
        document_texts: List[str],
        simulation_requirement: str,
        additional_context: Optional[str]
    ) -> str:
        combined_text = "\n\n---\n\n".join(document_texts)
        if len(combined_text) > self.MAX_TEXT_LENGTH_FOR_LLM:
            combined_text = combined_text[:self.MAX_TEXT_LENGTH_FOR_LLM]

        message = f"""Simulation requirement:
{simulation_requirement}

Source text:
{combined_text}
"""

        if additional_context:
            message += f"""

{additional_context}
"""

        message += """
Design entity and relationship types for a social-media opinion simulation.

Requirements:
1. Output exactly 10 entity types
2. The last 2 must be `Person` and `Organization`
3. The first 8 must be concrete text-derived types
4. Entity types must represent real actors rather than abstract concepts
5. Do not use reserved attribute names such as `name`, `uuid`, or `group_id`; prefer `full_name`, `org_name`, and similar alternatives
"""

        return message
    
    def _validate_and_process(self, result: Dict[str, Any]) -> Dict[str, Any]:
        
        if "entity_types" not in result:
            result["entity_types"] = []
        if "edge_types" not in result:
            result["edge_types"] = []
        if "analysis_summary" not in result:
            result["analysis_summary"] = ""

        # Coerce list elements to objects (LLM sometimes returns strings or wrong shapes)
        entity_types: List[Dict[str, Any]] = []
        for ent in result["entity_types"]:
            if isinstance(ent, str) and ent.strip():
                pascal = _to_pascal_case(ent)
                entity_types.append(
                    {
                        "name": pascal,
                        "description": f"A {pascal} entity type derived from the source text.",
                        "attributes": [],
                        "examples": [],
                    }
                )
            elif isinstance(ent, dict):
                entity_types.append(ent)
            else:
                logger.warning(f"Skipping invalid entity_types entry: {type(ent).__name__}")
        result["entity_types"] = entity_types

        edge_types: List[Dict[str, Any]] = []
        for ed in result["edge_types"]:
            if isinstance(ed, str) and ed.strip():
                n = re.sub(r"\s+", "_", ed.strip()).upper()
                edge_types.append(
                    {
                        "name": n,
                        "description": f"Relationship: {n}",
                        "attributes": [],
                        "source_targets": [],
                    }
                )
            elif isinstance(ed, dict):
                edge_types.append(ed)
            else:
                logger.warning(f"Skipping invalid edge_types entry: {type(ed).__name__}")
        result["edge_types"] = edge_types

        for entity in result["entity_types"]:
            attrs_in = entity.get("attributes") or []
            norm_attrs: List[Dict[str, str]] = []
            if isinstance(attrs_in, list):
                for a in attrs_in:
                    na = _normalize_attr_item(a)
                    if na:
                        norm_attrs.append(na)
            entity["attributes"] = norm_attrs
            if "examples" not in entity:
                entity["examples"] = []
            if not isinstance(entity.get("description", ""), str):
                entity["description"] = str(entity.get("description", ""))
            if len(entity.get("description", "")) > 100:
                entity["description"] = entity["description"][:97] + "..."

        for edge in result["edge_types"]:
            st_in = edge.get("source_targets") or []
            norm_st: List[Dict[str, str]] = []
            if isinstance(st_in, list):
                for st in st_in:
                    ns = _normalize_source_target(st)
                    if ns:
                        norm_st.append(ns)
            edge["source_targets"] = norm_st
            e_attrs = edge.get("attributes") or []
            norm_ea: List[Dict[str, str]] = []
            if isinstance(e_attrs, list):
                for a in e_attrs:
                    na = _normalize_attr_item(a)
                    if na:
                        norm_ea.append(na)
            edge["attributes"] = norm_ea
            if not isinstance(edge.get("description", ""), str):
                edge["description"] = str(edge.get("description", ""))
            if len(edge.get("description", "")) > 100:
                edge["description"] = edge["description"][:97] + "..."
        
        entity_name_map = {}
        for entity in result["entity_types"]:
            if "name" in entity:
                original_name = entity["name"]
                entity["name"] = _to_pascal_case(str(original_name))
                if entity["name"] != original_name:
                    logger.warning(f"Entity type name '{original_name}' auto-converted to '{entity['name']}'")
                entity_name_map[original_name] = entity["name"]
            if "attributes" not in entity:
                entity["attributes"] = []
            if "examples" not in entity:
                entity["examples"] = []
            if len(entity.get("description", "")) > 100:
                entity["description"] = entity["description"][:97] + "..."
        
        for edge in result["edge_types"]:
            if "name" in edge:
                original_name = edge["name"]
                edge["name"] = str(original_name).upper()
                if edge["name"] != original_name:
                    logger.warning(f"Edge type name '{original_name}' auto-converted to '{edge['name']}'")
            for st in edge.get("source_targets", []):
                if not isinstance(st, dict):
                    continue
                if st.get("source") in entity_name_map:
                    st["source"] = entity_name_map[st["source"]]
                if st.get("target") in entity_name_map:
                    st["target"] = entity_name_map[st["target"]]
            if "source_targets" not in edge:
                edge["source_targets"] = []
            if "attributes" not in edge:
                edge["attributes"] = []
            if len(edge.get("description", "")) > 100:
                edge["description"] = edge["description"][:97] + "..."
        
        MAX_ENTITY_TYPES = 10
        MAX_EDGE_TYPES = 10

        seen_names = set()
        deduped = []
        for entity in result["entity_types"]:
            name = entity.get("name", "")
            if name and name not in seen_names:
                seen_names.add(name)
                deduped.append(entity)
            elif name in seen_names:
                logger.warning(f"Duplicate entity type '{name}' removed during validation")
        result["entity_types"] = deduped

        person_fallback = {
            "name": "Person",
            "description": "Any individual person not fitting other specific person types.",
            "attributes": [
                {"name": "full_name", "type": "text", "description": "Full name of the person"},
                {"name": "role", "type": "text", "description": "Role or occupation"}
            ],
            "examples": ["ordinary citizen", "anonymous netizen"]
        }
        
        organization_fallback = {
            "name": "Organization",
            "description": "Any organization not fitting other specific organization types.",
            "attributes": [
                {"name": "org_name", "type": "text", "description": "Name of the organization"},
                {"name": "org_type", "type": "text", "description": "Type of organization"}
            ],
            "examples": ["small business", "community group"]
        }
        
        entity_names = {e["name"] for e in result["entity_types"]}
        has_person = "Person" in entity_names
        has_organization = "Organization" in entity_names
        
        fallbacks_to_add = []
        if not has_person:
            fallbacks_to_add.append(person_fallback)
        if not has_organization:
            fallbacks_to_add.append(organization_fallback)
        
        if fallbacks_to_add:
            current_count = len(result["entity_types"])
            needed_slots = len(fallbacks_to_add)
            
            if current_count + needed_slots > MAX_ENTITY_TYPES:
                to_remove = current_count + needed_slots - MAX_ENTITY_TYPES
                result["entity_types"] = result["entity_types"][:-to_remove]
            
            result["entity_types"].extend(fallbacks_to_add)
        
        if len(result["entity_types"]) > MAX_ENTITY_TYPES:
            result["entity_types"] = result["entity_types"][:MAX_ENTITY_TYPES]
        
        if len(result["edge_types"]) > MAX_EDGE_TYPES:
            result["edge_types"] = result["edge_types"][:MAX_EDGE_TYPES]
        
        return result
    
    def generate_python_code(self, ontology: Dict[str, Any]) -> str:
        """
        
        Args:
            
        Returns:
        """
        code_lines = [
            '"""',
            'Custom entity type definitions',
            'Auto-generated by go-mirofish for social opinion simulation',
            '"""',
            '',
            'from pydantic import Field',
            'from zep_cloud.external_clients.ontology import EntityModel, EntityText, EdgeModel',
            '',
            '',
            '# ============== Entity type definitions ==============',
            '',
        ]
        
        for entity in ontology.get("entity_types", []):
            name = entity["name"]
            desc = entity.get("description", f"A {name} entity.")
            
            code_lines.append(f'class {name}(EntityModel):')
            code_lines.append(f'    """{desc}"""')
            
            attrs = entity.get("attributes", [])
            if attrs:
                for attr in attrs:
                    attr_name = attr["name"]
                    attr_desc = attr.get("description", attr_name)
                    code_lines.append(f'    {attr_name}: EntityText = Field(')
                    code_lines.append(f'        description="{attr_desc}",')
                    code_lines.append(f'        default=None')
                    code_lines.append(f'    )')
            else:
                code_lines.append('    pass')
            
            code_lines.append('')
            code_lines.append('')
        
        code_lines.append('# ============== Relationship type definitions ==============')
        code_lines.append('')
        
        for edge in ontology.get("edge_types", []):
            name = edge["name"]
            class_name = ''.join(word.capitalize() for word in name.split('_'))
            desc = edge.get("description", f"A {name} relationship.")
            
            code_lines.append(f'class {class_name}(EdgeModel):')
            code_lines.append(f'    """{desc}"""')
            
            attrs = edge.get("attributes", [])
            if attrs:
                for attr in attrs:
                    attr_name = attr["name"]
                    attr_desc = attr.get("description", attr_name)
                    code_lines.append(f'    {attr_name}: EntityText = Field(')
                    code_lines.append(f'        description="{attr_desc}",')
                    code_lines.append(f'        default=None')
                    code_lines.append(f'    )')
            else:
                code_lines.append('    pass')
            
            code_lines.append('')
            code_lines.append('')
        
        code_lines.append('# ============== Type configuration ==============')
        code_lines.append('')
        code_lines.append('ENTITY_TYPES = {')
        for entity in ontology.get("entity_types", []):
            name = entity["name"]
            code_lines.append(f'    "{name}": {name},')
        code_lines.append('}')
        code_lines.append('')
        code_lines.append('EDGE_TYPES = {')
        for edge in ontology.get("edge_types", []):
            name = edge["name"]
            class_name = ''.join(word.capitalize() for word in name.split('_'))
            code_lines.append(f'    "{name}": {class_name},')
        code_lines.append('}')
        code_lines.append('')
        
        code_lines.append('EDGE_SOURCE_TARGETS = {')
        for edge in ontology.get("edge_types", []):
            name = edge["name"]
            source_targets = edge.get("source_targets", [])
            if source_targets:
                st_list = ', '.join([
                    f'{{"source": "{st.get("source", "Entity")}", "target": "{st.get("target", "Entity")}"}}'
                    for st in source_targets
                ])
                code_lines.append(f'    "{name}": [{st_list}],')
        code_lines.append('}')
        
        return '\n'.join(code_lines)
