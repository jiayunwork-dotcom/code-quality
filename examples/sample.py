# 示例Python代码，用于测试代码质量工具
import os
import sys
from datetime import datetime


class UserService:
    """用户服务类"""
    
    def __init__(self, db_connection):
        self.db_connection = db_connection
        self.cache = {}
        self.logger = None
        self.config = {}
        self.metrics = {}
        self.queue = []
        self.validator = None
    
    def get_user(self, user_id, include_details=False, fetch_related=False, 
                 use_cache=True, timeout=30, retry_count=3):
        """获取用户信息 - 长参数列表"""
        if not user_id:
            return None
        
        if use_cache and user_id in self.cache:
            return self.cache[user_id]
        
        user = self.db_connection.query("SELECT * FROM users WHERE id = ?", user_id)
        
        if user is None:
            return None
        
        if include_details:
            user['profile'] = self.get_user_profile(user_id)
            user['preferences'] = self.get_user_preferences(user_id)
            user['settings'] = self.get_user_settings(user_id)
            user['permissions'] = self.get_user_permissions(user_id)
            user['roles'] = self.get_user_roles(user_id)
            user['groups'] = self.get_user_groups(user_id)
        
        if fetch_related:
            user['orders'] = self.get_user_orders(user_id)
            user['transactions'] = self.get_user_transactions(user_id)
            user['notifications'] = self.get_user_notifications(user_id)
        
        if use_cache:
            self.cache[user_id] = user
        
        return user
    
    def complex_function(self, data, config, options, params, settings):
        """高复杂度函数示例"""
        result = []
        
        for item in data:
            if item is None:
                continue
            
            if 'id' not in item:
                if config.get('strict_mode', False):
                    raise ValueError("Missing id")
                else:
                    continue
            
            item_id = item['id']
            
            if item_id in options.get('exclude', []):
                continue
            
            processed = {}
            
            if options.get('process_fields', False):
                for field, value in item.items():
                    if field in params.get('allowed_fields', []):
                        if isinstance(value, str):
                            if settings.get('trim_strings', True):
                                value = value.strip()
                            if settings.get('lowercase', False):
                                value = value.lower()
                        processed[field] = value
            
            if options.get('validate', False):
                valid = True
                if params.get('min_length', 0) > 0:
                    if len(processed) < params['min_length']:
                        valid = False
                if params.get('max_length', 100) > 0:
                    if len(processed) > params['max_length']:
                        valid = False
                if not valid and config.get('skip_invalid', False):
                    continue
            
            if options.get('transform', False):
                transform_type = params.get('transform_type', 'default')
                if transform_type == 'uppercase':
                    for k, v in processed.items():
                        if isinstance(v, str):
                            processed[k] = v.upper()
                elif transform_type == 'flatten':
                    flattened = {}
                    for k, v in processed.items():
                        if isinstance(v, dict):
                            for k2, v2 in v.items():
                                flattened[f"{k}_{k2}"] = v2
                        else:
                            flattened[k] = v
                    processed = flattened
            
            if config.get('add_timestamps', False):
                processed['created_at'] = datetime.now().isoformat()
            
            if options.get('filter', False):
                if params.get('filter_field') in processed:
                    filter_value = processed[params['filter_field']]
                    if filter_value not in params.get('allowed_values', []):
                        continue
            
            result.append(processed)
        
        if options.get('sort', False):
            sort_field = params.get('sort_field', 'id')
            reverse = params.get('sort_reverse', False)
            result.sort(key=lambda x: x.get(sort_field, ''), reverse=reverse)
        
        return result
    
    def get_user_profile(self, user_id):
        return {}
    
    def get_user_preferences(self, user_id):
        return {}
    
    def get_user_settings(self, user_id):
        return {}
    
    def get_user_permissions(self, user_id):
        return []
    
    def get_user_roles(self, user_id):
        return []
    
    def get_user_groups(self, user_id):
        return []
    
    def get_user_orders(self, user_id):
        return []
    
    def get_user_transactions(self, user_id):
        return []
    
    def get_user_notifications(self, user_id):
        return []
